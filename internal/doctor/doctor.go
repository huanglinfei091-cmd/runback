// Package doctor performs read-only Alpha readiness checks. It never repairs
// Docker, creates networks, or executes repository workflows.
package doctor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/huanglinfei091-cmd/runback/internal/acquire"
	"github.com/huanglinfei091-cmd/runback/internal/online"
)

const runnerImage = "ghcr.io/catthehacker/ubuntu:act-latest"

type commandRunner func(context.Context, []string, string, ...string) (string, error)

type checker struct {
	lookPath  func(string) (string, error)
	run       commandRunner
	access    func(context.Context) (online.AccessStatus, error)
	workspace func() (string, error)
	diskFree  func(string) (uint64, error)
	pathState func() (bool, string)
}

// Run prints a concise environment report and returns whether required Alpha
// dependencies are available. Warnings do not trigger host mutations.
func Run(ctx context.Context, out io.Writer) bool {
	c := checker{
		lookPath:  exec.LookPath,
		run:       runCommand,
		access:    online.ProbeAccess,
		workspace: acquire.AllocateWorkspace,
		diskFree:  availableBytes,
		pathState: executableOnPath,
	}
	return c.check(ctx, out)
}

func (c checker) check(ctx context.Context, out io.Writer) bool {
	fmt.Fprintln(out, "RunBack Doctor")
	fmt.Fprintln(out)

	home, err := os.MkdirTemp("", "runback-doctor-")
	if err != nil {
		fmt.Fprintln(out, "✗ Diagnostic workspace")
		fmt.Fprintln(out, "RunBack is not ready.")
		return false
	}
	defer os.RemoveAll(home)
	env := subprocessEnv(home)

	ready := true
	found := map[string]bool{}
	for _, tool := range []string{"git", "docker", "act"} {
		if _, err := c.lookPath(tool); err != nil {
			fmt.Fprintf(out, "✗ %s\n", displayName(tool))
			printProblem(out, displayName(tool)+" was not found on PATH.", toolCause(tool), toolNext(tool))
			found[tool] = false
			ready = false
			continue
		}
		found[tool] = true
		version, err := c.run(ctx, env, tool, "--version")
		if err != nil {
			fmt.Fprintf(out, "✗ %s\n", displayName(tool))
			printProblem(out, displayName(tool)+" could not report its version.", "The executable was found but could not run successfully.", "Reinstall "+displayName(tool)+", then run: runback doctor")
			ready = false
			continue
		}
		fmt.Fprintf(out, "✓ %s (%s)\n", displayName(tool), firstLine(version))
	}

	daemonReady := false
	if found["docker"] {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		version, err := c.run(probeCtx, env, "docker", "info", "--format", "{{.ServerVersion}}")
		cancel()
		if err != nil {
			fmt.Fprintln(out, "✗ Docker daemon")
			printProblem(out, "RunBack cannot reach the Docker daemon.", "Docker is installed, but its daemon is stopped or inaccessible to this user.", "docker info")
			ready = false
		} else {
			daemonReady = true
			fmt.Fprintf(out, "✓ Docker daemon (%s)\n", firstLine(version))
		}
	}

	accessCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	access, accessErr := c.access(accessCtx)
	cancel()
	if accessErr != nil {
		var onlineErr *online.Error
		if errors.As(accessErr, &onlineErr) {
			fmt.Fprintf(out, "✗ GitHub API (%s, %s)\n", modeName(access.Mode), onlineErr.Cause)
		} else {
			fmt.Fprintf(out, "✗ GitHub API (%s)\n", modeName(access.Mode))
		}
		printProblem(out, "GitHub evidence acquisition is unavailable.", "The API probe failed before a public run could be acquired.", "curl -I https://api.github.com/rate_limit")
		ready = false
	} else {
		if access.Mode == "anonymous" {
			fmt.Fprintln(out, "! GitHub token not configured; public repositories use the lower anonymous API limit.")
			printProblem(out, "Anonymous GitHub requests have a lower quota.", "No RUNBACK_GITHUB_TOKEN or GH_TOKEN is set for this process.", `RUNBACK_GITHUB_TOKEN="$(gh auth token)" runback doctor`)
		}
		if access.Remaining <= 0 {
			fmt.Fprintf(out, "✗ GitHub API quota exhausted (%s; reset %s)\n", modeName(access.Mode), resetName(access.Reset))
			printProblem(out, "The GitHub API quota is exhausted.", "RunBack cannot acquire immutable run evidence until quota is available.", `RUNBACK_GITHUB_TOKEN="$(gh auth token)" runback doctor`)
			ready = false
		} else {
			fmt.Fprintf(out, "✓ GitHub API (%s, %d/%d remaining)\n", modeName(access.Mode), access.Remaining, access.Limit)
		}
	}

	if err := checkWorkspace(c.workspace); err != nil {
		fmt.Fprintln(out, "✗ Replay workspace")
		printProblem(out, "RunBack cannot create its short replay workspace.", "The temporary directory is unavailable or not writable.", "ls -ld /tmp /tmp/rb && df -h /tmp")
		ready = false
	} else {
		fmt.Fprintln(out, "✓ Replay workspace (/tmp/rb/<short-id>)")
	}

	if c.diskFree != nil {
		free, diskErr := c.diskFree(os.TempDir())
		switch {
		case diskErr != nil:
			fmt.Fprintln(out, "! Disk space check unavailable")
			printProblem(out, "Free space could not be measured.", "RunBack did not modify the filesystem.", "df -h /tmp")
		case free < 2<<30:
			fmt.Fprintf(out, "✗ Disk space (%.1f GiB free)\n", gibibytes(free))
			printProblem(out, "Less than 2 GiB is free for replay data.", "Images, source, actions and dependencies can exhaust the temporary filesystem.", "df -h /tmp")
			ready = false
		case free < 5<<30:
			fmt.Fprintf(out, "! Disk space is low (%.1f GiB free)\n", gibibytes(free))
			printProblem(out, "Less than 5 GiB is free for replay data.", "Larger runner images or dependency downloads may fail.", "df -h /tmp")
		default:
			fmt.Fprintf(out, "✓ Disk space (%.1f GiB free)\n", gibibytes(free))
		}
	}

	if c.pathState != nil {
		onPath, _ := c.pathState()
		if onPath {
			fmt.Fprintln(out, "✓ RunBack on PATH")
		} else {
			fmt.Fprintln(out, "! RunBack is not on PATH")
			printProblem(out, "A new shell may not find the runback command.", "The running binary's directory is absent from PATH.", `export PATH="$HOME/.local/bin:$PATH"`)
		}
	}

	if daemonReady {
		checkNetwork(ctx, out, c.run, env)
	} else {
		fmt.Fprintln(out, "! Docker networking check skipped because the daemon is unavailable.")
	}

	fmt.Fprintln(out)
	if ready {
		fmt.Fprintln(out, "Ready to reproduce public GitHub Actions failures.")
	} else {
		fmt.Fprintln(out, "RunBack is not ready. Resolve the failed checks above and run doctor again.")
	}
	return ready
}

func runCommand(ctx context.Context, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func subprocessEnv(home string) []string {
	allowed := map[string]bool{
		"PATH": true, "SYSTEMROOT": true, "WINDIR": true,
		"TEMP": true, "TMP": true, "TMPDIR": true, "DOCKER_HOST": true,
		"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true,
		"http_proxy": true, "https_proxy": true, "no_proxy": true,
	}
	var env []string
	for _, item := range os.Environ() {
		name := strings.SplitN(item, "=", 2)[0]
		if allowed[name] {
			env = append(env, item)
		}
	}
	return append(env,
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"),
		"GIT_TERMINAL_PROMPT=0",
	)
}

func checkWorkspace(allocate func() (string, error)) error {
	root, err := allocate()
	if err != nil {
		return err
	}
	probe := filepath.Join(root, ".doctor-write-probe")
	if err = os.WriteFile(probe, []byte("runback"), 0600); err != nil {
		return err
	}
	if err = os.Remove(probe); err != nil {
		return err
	}
	return os.Remove(root)
}

func checkNetwork(ctx context.Context, out io.Writer, run commandRunner, env []string) {
	networkRun := func(args ...string) (string, error) {
		probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		return run(probeCtx, env, "docker", args...)
	}
	if _, err := networkRun("image", "inspect", runnerImage, "--format", "{{.Id}}"); err != nil {
		fmt.Fprintln(out, "! Docker networking check deferred; the default replay image is not present locally.")
		printProblem(out, "Container networking was not probed.", "Doctor does not pull a large runner image as a side effect.", "runback <failed-github-actions-run-url>")
		return
	}
	probe := []string{"run", "--rm", "--network", "bridge", "--entrypoint", "bash", runnerImage, "-c", "timeout 8 bash -c 'exec 3<>/dev/tcp/github.com/443'"}
	if _, err := networkRun(probe...); err == nil {
		fmt.Fprintln(out, "✓ Docker networking (default bridge)")
		return
	}
	list, err := networkRun("network", "ls", "--filter", "label=io.runback.managed=true", "--format", "{{.Name}}")
	if err == nil {
		names := strings.Fields(list)
		if len(names) > 8 {
			names = names[:8]
		}
		for _, name := range names {
			properties, inspectErr := networkRun("network", "inspect", name, "--format", `{{.Driver}}|{{.Internal}}|{{index .Labels "io.runback.managed"}}`)
			if inspectErr != nil || strings.TrimSpace(properties) != "bridge|false|true" {
				continue
			}
			managedProbe := append([]string{}, probe...)
			managedProbe[3] = name
			if _, probeErr := networkRun(managedProbe...); probeErr == nil {
				fmt.Fprintln(out, "✓ Docker networking (RunBack-managed fallback available)")
				return
			}
		}
	}
	findings := "! Docker default bridge probe failed; replay may attempt one RunBack-managed fallback."
	fmt.Fprintln(out, findings)
	printProblem(out, "A container could not reach github.com over the default bridge.", "Action or dependency downloads may fail during replay.", "docker network inspect bridge")
}

func printProblem(out io.Writer, problem, cause, next string) {
	fmt.Fprintf(out, "  Problem: %s\n  Cause: %s\n  Next: %s\n", problem, cause, next)
}

func toolCause(name string) string {
	switch name {
	case "git":
		return "RunBack needs Git to fetch the historical commit."
	case "docker":
		return "RunBack uses Docker to isolate the act replay."
	default:
		return "RunBack delegates GitHub Actions workflow execution to act."
	}
}

func toolNext(name string) string {
	switch name {
	case "git":
		return "sudo apt-get update && sudo apt-get install -y git"
	case "docker":
		return "Install Docker Engine from https://docs.docker.com/engine/install/, then run: runback doctor"
	default:
		return "Install act from https://nektosact.com/installation/index.html, then run: runback doctor"
	}
}

func gibibytes(bytes uint64) float64 { return float64(bytes) / float64(uint64(1)<<30) }

func executableOnPath() (bool, string) {
	executable, err := os.Executable()
	if err != nil {
		return false, ""
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		resolved = executable
	}
	for _, directory := range filepath.SplitList(os.Getenv("PATH")) {
		candidate := filepath.Join(directory, filepath.Base(executable))
		candidateResolved, candidateErr := filepath.EvalSymlinks(candidate)
		if candidateErr == nil && filepath.Clean(candidateResolved) == filepath.Clean(resolved) {
			return true, filepath.Dir(resolved)
		}
	}
	return false, filepath.Dir(resolved)
}

func displayName(name string) string {
	if name == "git" {
		return "Git"
	}
	if name == "docker" {
		return "Docker"
	}
	return "act"
}

func modeName(mode string) string {
	if mode == "token" {
		return "authenticated"
	}
	if mode == "anonymous" {
		return "anonymous"
	}
	return "unknown mode"
}

func resetName(reset string) string {
	if reset == "" {
		return "unknown"
	}
	return reset
}

func firstLine(value string) string {
	line := strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
	if len(line) > 100 {
		return line[:100]
	}
	if line == "" {
		return "available"
	}
	return line
}
