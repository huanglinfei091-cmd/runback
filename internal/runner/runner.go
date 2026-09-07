package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/failure"
	"github.com/huanglinfei091-cmd/runback/internal/fingerprint"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Result struct {
	Stage           string                 `json:"stage"`
	Executor        string                 `json:"executor"`
	Mode            string                 `json:"mode"`
	Cause           string                 `json:"cause,omitempty"`
	TargetSucceeded bool                   `json:"target_succeeded"`
	TargetReached   bool                   `json:"target_reached"`
	Evidence        failure.Match          `json:"evidence"`
	Status          string                 `json:"status"`
	Started         string                 `json:"started_at"`
	Duration        float64                `json:"duration_seconds"`
	ExitCode        int                    `json:"exit_code"`
	Comparison      fingerprint.Comparison `json:"comparison"`
	Error           string                 `json:"error,omitempty"`
	Command         []string               `json:"command"`
	ImageID         string                 `json:"image_id,omitempty"`
	ContainerIDs    []string               `json:"container_ids,omitempty"`
	WorkspaceDirty  bool                   `json:"workspace_dirty"`
}

func env(home string) []string {
	allowed := map[string]bool{"PATH": true, "SYSTEMROOT": true, "WINDIR": true, "TEMP": true, "TMP": true, "TMPDIR": true, "DOCKER_HOST": true, "HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true, "http_proxy": true, "https_proxy": true, "no_proxy": true}
	out := []string{}
	for _, s := range os.Environ() {
		if allowed[strings.SplitN(s, "=", 2)[0]] {
			out = append(out, s)
		}
	}
	return append(out, "HOME="+home, "USERPROFILE="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"), "GIT_TERMINAL_PROMPT=0")
}
func command(ctx context.Context, p replay.Plan, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = p.Root
	cmd.Env = env(p.Home)
	return cmd
}
func run(ctx context.Context, p replay.Plan, name string, args ...string) (string, error) {
	b, e := command(ctx, p, name, args...).CombinedOutput()
	if e != nil {
		return string(b), fmt.Errorf("%s failed: %w: %.1500s", name, e, b)
	}
	return strings.TrimSpace(string(b)), nil
}
func Checkout(ctx context.Context, p replay.Plan, l lockfile.Lock) error {
	if _, e := os.Stat(filepath.Join(p.Checkout, ".git")); os.IsNotExist(e) {
		if _, e = run(ctx, p, "git", "init", p.Checkout); e != nil {
			return e
		}
		if _, e = run(ctx, p, "git", "-C", p.Checkout, "remote", "add", "origin", "https://github.com/"+l.Repository+".git"); e != nil {
			return e
		}
		if _, e = run(ctx, p, "git", "-C", p.Checkout, "-c", "core.hooksPath="+p.Home, "fetch", "--depth=1", "origin", l.Commit); e != nil {
			return e
		}
		if _, e = run(ctx, p, "git", "-C", p.Checkout, "-c", "core.hooksPath="+p.Home, "checkout", "--detach", l.Commit); e != nil {
			return e
		}
	}
	head, e := run(ctx, p, "git", "-C", p.Checkout, "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	if head != l.Commit {
		return fmt.Errorf("workspace HEAD differs from locked commit; refusing to reset user changes")
	}
	return nil
}
func Doctor(ctx context.Context, w io.Writer) bool {
	ok := true
	for _, name := range []string{"git", "docker", "act"} {
		path, e := exec.LookPath(name)
		if e != nil {
			fmt.Fprintf(w, "MISSING  %s\n", name)
			ok = false
		} else {
			fmt.Fprintf(w, "OK       %s: %s\n", name, path)
		}
	}
	if _, e := exec.LookPath("docker"); e == nil {
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		home, e := os.MkdirTemp("", "runback-doctor-")
		if e != nil {
			cancel()
			fmt.Fprintln(w, "MISSING  cannot prepare isolated Docker diagnostic environment")
			return false
		}
		defer os.RemoveAll(home)
		probe := exec.CommandContext(c, "docker", "info", "--format", "{{.OSType}} / {{.ServerVersion}}")
		probe.Env = env(home)
		b, e := probe.CombinedOutput()
		cancel()
		if e != nil {
			fmt.Fprintf(w, "MISSING  Docker daemon: %s\n", b)
			ok = false
		} else {
			fmt.Fprintf(w, "OK       Docker %s", b)
		}
	}
	fmt.Fprintf(w, "HOST     %s/%s\nAUTH     GitHub acquisition: RUNBACK_GITHUB_TOKEN > GH_TOKEN > anonymous; execution is isolated\n", runtime.GOOS, runtime.GOARCH)
	return ok
}
func Execute(ctx context.Context, p replay.Plan, l lockfile.Lock, out io.Writer) (Result, error) {
	started := time.Now()
	r := Result{Stage: "PREPARE", Executor: "act", Mode: "FULL_JOB", Status: "REPLAY_BLOCKED", Started: started.UTC().Format(time.RFC3339), ExitCode: -1, Command: append([]string{"act"}, p.Args...)}
	r.Evidence = failure.Compare(l.Failure.Log, "", l.Failure.Name, l.Failure.Command, -1, false)
	finish := func(err error) (Result, error) {
		r.Duration = time.Since(started).Seconds()
		if err != nil {
			r.Error = err.Error()
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		if save := lockfile.Atomic(filepath.Join(p.Root, "result.json"), b); save != nil && err == nil {
			err = save
		}
		return r, err
	}
	if len(l.Blockers) > 0 {
		r.Cause = strings.Split(l.Blockers[0], ":")[0]
		return finish(fmt.Errorf("replay blocked: %s", strings.Join(l.Blockers, "; ")))
	}
	if !Doctor(ctx, out) {
		r.Cause = "ENVIRONMENT_MISMATCH"
		return finish(fmt.Errorf("install missing dependencies; runback doctor"))
	}
	if e := Checkout(ctx, p, l); e != nil {
		r.Cause = "NETWORK_DEPENDENCY"
		return finish(e)
	}
	dirty, e := run(ctx, p, "git", "-C", p.Checkout, "status", "--porcelain")
	if e != nil {
		return finish(e)
	}
	r.WorkspaceDirty = dirty != ""
	file, e := os.Create(filepath.Join(p.Root, "local.jsonl"))
	if e != nil {
		return finish(e)
	}
	defer file.Close()
	cmd := command(ctx, p, "act", p.Args...)
	cmd.Stdout = io.MultiWriter(file, out)
	cmd.Stderr = io.MultiWriter(file, out)
	r.Stage = "EXECUTE"
	e = cmd.Run()
	r.ExitCode = 0
	if e != nil {
		r.ExitCode = -1
		if ex, ok := e.(*exec.ExitError); ok {
			r.ExitCode = ex.ExitCode()
		}
	}
	file.Close()
	if ctx.Err() != nil {
		r.Status = "REPLAY_BLOCKED"
		r.Cause = "EXECUTION_TIMEOUT"
		return finish(fmt.Errorf("replay interrupted: %w", ctx.Err()))
	}
	imageID, _ := run(ctx, p, "docker", "image", "inspect", l.Image, "--format", "{{.Id}}")
	r.ImageID = imageID
	owned, _ := run(ctx, p, "docker", "ps", "-aq", "--no-trunc", "--filter", "label=io.runback.case="+l.Key())
	r.ContainerIDs = strings.Fields(owned)
	log, e2 := os.ReadFile(filepath.Join(p.Root, "local.jsonl"))
	if e2 != nil {
		return finish(e2)
	}
	local, failed := ActStepLog(string(log), l)
	r.Comparison = fingerprint.Compare(l.Failure.Log, local, r.ExitCode, failed)
	r.Evidence = failure.Compare(l.Failure.Log, local, l.Failure.Name, l.Failure.Command, r.ExitCode, failed)
	r.TargetReached = len(local) > 0
	r.TargetSucceeded = ActStepSucceeded(string(log), l)
	r.Status = r.Evidence.Status
	if r.TargetReached {
		r.Stage = "MATCH"
	}
	if r.Status == "REPLAY_BLOCKED" {
		r.Cause = ExecutorCause(string(log))
	}
	if r.ExitCode == -1 {
		return finish(e)
	}
	return finish(nil)
}

// ActStepLog uses act JSON step identity; output from setup/other steps cannot match by accident.
func ActStepLog(log string, l lockfile.Lock) (string, bool) {
	var lines []string
	failed := false
	scanner := bufio.NewScanner(strings.NewReader(log))
	scanner.Buffer(make([]byte, 65536), 2<<20)
	for scanner.Scan() {
		var v map[string]any
		if json.Unmarshal(scanner.Bytes(), &v) != nil {
			continue
		}
		msg, _ := v["msg"].(string)
		step := fmt.Sprint(v["step"])
		job := fmt.Sprint(v["jobID"])
		if job != "<nil>" && job != l.JobID {
			continue
		}
		selected := step == l.Failure.Name || (strings.HasPrefix(l.Failure.Name, "Run ") && step == strings.TrimPrefix(l.Failure.Name, "Run "))
		stage, _ := v["stage"].(string)
		selected = selected && (stage == "" || strings.EqualFold(stage, "Main"))
		if selected && strings.Contains(msg, "Failure - Main ") {
			failed = true
		}
		if selected {
			lines = append(lines, msg)
			if strings.Contains(msg, "Failure - Main") {
				failed = true
			}
		}
	}
	return strings.Join(lines, "\n"), failed
}
func Shell(ctx context.Context, p replay.Plan) error {
	b, e := os.ReadFile(filepath.Join(p.Root, "result.json"))
	if e != nil {
		return fmt.Errorf("run replay first: %w", e)
	}
	var r Result
	if e = json.Unmarshal(b, &r); e != nil {
		return e
	}
	if len(r.ContainerIDs) != 1 {
		return fmt.Errorf("expected one retained replay container, found %d; inspect result.json", len(r.ContainerIDs))
	}
	id := r.ContainerIDs[0]
	if len(id) != 64 || strings.Trim(id, "0123456789abcdef") != "" {
		return fmt.Errorf("invalid saved container ID")
	}
	if _, e = run(ctx, p, "docker", "start", id); e != nil {
		return e
	}
	cmd := command(ctx, p, "docker", "exec", "-it", id, "bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ActStepSucceeded requires an explicit act success event for the original target.
// A skipped target with log output cannot establish a verified fix.
func ActStepSucceeded(log string, l lockfile.Lock) bool {
	for _, line := range strings.Split(log, "\n") {
		var v map[string]any
		if json.Unmarshal([]byte(line), &v) != nil {
			continue
		}
		job := fmt.Sprint(v["jobID"])
		if job != "<nil>" && job != l.JobID {
			continue
		}
		step := fmt.Sprint(v["step"])
		selected := step == l.Failure.Name || (strings.HasPrefix(l.Failure.Name, "Run ") && step == strings.TrimPrefix(l.Failure.Name, "Run "))
		stage, _ := v["stage"].(string)
		if !selected || (stage != "" && !strings.EqualFold(stage, "Main")) {
			continue
		}
		msg, _ := v["msg"].(string)
		if strings.Contains(msg, "Success - Main ") {
			return true
		}
	}
	return false
}
