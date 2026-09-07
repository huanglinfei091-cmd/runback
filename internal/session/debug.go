package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/failure"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type DebugResult struct {
	Status          string        `json:"status"`
	Mode            string        `json:"mode"`
	Session         string        `json:"session"`
	Image           string        `json:"image"`
	Container       string        `json:"container"`
	Worktree        string        `json:"worktree"`
	Exit            int           `json:"exit"`
	DependencySetup bool          `json:"dependency_setup"`
	DependencyReset bool          `json:"dependency_reset"`
	NetworkDownload string        `json:"network_download"`
	Evidence        failure.Match `json:"evidence"`
}

func dependencyFile(path string) bool {
	n := filepath.Base(path)
	return n == "pyproject.toml" || n == "uv.lock" || n == "tox.ini" || n == "setup.py" || n == "setup.cfg" || n == "Pipfile" || n == "Pipfile.lock" || n == ".python-version" || strings.HasPrefix(n, "requirements") && strings.HasSuffix(n, ".txt")
}
func DependencyHash(ctx context.Context, s State) (string, error) {
	raw, e := output(ctx, s.Root, s.Worktree, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if e != nil {
		return "", e
	}
	files := strings.Split(raw, "\x00")
	sort.Strings(files)
	var parts []string
	for _, f := range files {
		if !dependencyFile(f) {
			continue
		}
		b, e := os.ReadFile(filepath.Join(s.Worktree, f))
		if os.IsNotExist(e) {
			parts = append(parts, f+":deleted")
			continue
		}
		if e != nil {
			return "", e
		}
		parts = append(parts, f+":"+lockfile.Digest(string(b)))
	}
	return lockfile.Digest(strings.Join(parts, "\n")), nil
}
func DockerArgs(s State, name string, interactive bool, command string) ([]string, error) {
	if s.DebugBlocker != "" {
		return nil, fmt.Errorf("%s", s.DebugBlocker)
	}
	if !strings.HasPrefix(s.ImageID, "sha256:") {
		return nil, fmt.Errorf("session has no verified image ID")
	}
	cwd := filepath.Clean(filepath.Join("/github/workspace", s.Lock.Failure.WorkingDirectory))
	if cwd != "/github/workspace" && !strings.HasPrefix(cwd, "/github/workspace/") {
		return nil, fmt.Errorf("step working directory escapes worktree")
	}
	network := s.Lock.Network
	if network == "" {
		network = "bridge"
	}
	args := []string{"run", "--rm", "--init", "--name", name, "--label", "io.runback.session=" + s.ID, "--platform", "linux/amd64", "--network", network,
		"--mount", "type=bind,src=" + s.Worktree + ",dst=/github/workspace",
		"--mount", "type=bind,src=" + s.Cache + ",dst=/runback/cache",
		"--mount", "type=bind,src=" + filepath.Join(s.Cache, "python") + ",dst=" + s.PythonRoot + ",readonly",
		"--mount", "type=bind,src=" + filepath.Join(s.Cache, "uv-bin") + ",dst=" + filepath.Dir(s.UVPath) + ",readonly",
		"--workdir", cwd}
	if interactive {
		args = append(args, "-it")
	}
	env := map[string]string{}
	for k, v := range s.Environment {
		env[k] = v
	}
	for k, v := range map[string]string{
		"PATH":            s.PythonRoot + "/bin:" + filepath.Dir(s.UVPath) + ":/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"LD_LIBRARY_PATH": s.PythonRoot + "/lib", "pythonLocation": s.PythonRoot,
		"UV_PYTHON": s.PythonRoot + "/bin/python3", "UV_PYTHON_INSTALL_DIR": "/runback/cache/python-downloads",
		"UV_CACHE_DIR": "/runback/cache/uv", "UV_LINK_MODE": "copy",
		"HOME": "/tmp/runback-home", "XDG_CACHE_HOME": "/runback/cache/xdg", "PIP_CACHE_DIR": "/runback/cache/pip",
		"GITHUB_WORKSPACE": "/github/workspace", "CI": "true", "GITHUB_ACTIONS": "true",
	} {
		env[k] = v
	}
	var keys []string
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--env", k+"="+env[k])
	}
	args = append(args, "--entrypoint", "/bin/bash", s.ImageID)
	if interactive {
		return append(args, "--noprofile", "--norc"), nil
	}
	// Bash matches the failed-step shell and never evaluates the command on the host.
	if s.Lock.Failure.Shell == "sh" {
		return append(args, "-c", "exec /bin/sh -e -c \"$1\"", "runback", command), nil
	}
	return append(args, "--noprofile", "--norc", "-eo", "pipefail", "-c", command), nil
}
func Debug(ctx context.Context, s *State, interactive bool, out io.Writer) (DebugResult, error) {
	r := DebugResult{Mode: "STEP_ONLY", Session: s.ID, Image: s.ImageID, Worktree: s.Worktree, Exit: -1, NetworkDownload: "UNKNOWN"}
	if interactive {
		r.Mode = "DEV"
	}
	release, e := Acquire(*s)
	if e != nil {
		return r, e
	}
	defer release()
	hash, e := DependencyHash(ctx, *s)
	if e != nil {
		return r, e
	}
	if s.DependencyHash != "" && hash != s.DependencyHash {
		for _, n := range []string{".venv", ".tox"} {
			if e = os.RemoveAll(filepath.Join(s.Worktree, n)); e != nil {
				return r, e
			}
		}
		r.DependencyReset = true
	}
	// The original uv/tox command retains its sync/setup semantics on every replay.
	r.DependencySetup = s.DependencyHash == "" || r.DependencyReset
	r.Container = fmt.Sprintf("runback-debug-%d", time.Now().UnixNano())
	args, e := DockerArgs(*s, r.Container, interactive, s.Lock.Failure.Command)
	if e != nil {
		return r, e
	}
	logs := filepath.Join(s.Root, "logs")
	if e = os.MkdirAll(logs, 0700); e != nil {
		return r, e
	}
	path := filepath.Join(logs, r.Container+".log")
	f, e := os.Create(path)
	if e != nil {
		return r, e
	}
	defer f.Close()
	c := exec.CommandContext(ctx, "docker", args...)
	c.Env = CleanEnv(s.Root)
	c.Stdout = io.MultiWriter(out, f)
	c.Stderr = io.MultiWriter(out, f)
	if interactive {
		c.Stdin = os.Stdin
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		run(cleanup, s.Root, s.Root, "docker", "rm", "-f", r.Container)
	}()
	fmt.Fprintf(out, "Session: %s\nWorktree: %s\nMode: %s\n", s.ID, s.Worktree, r.Mode)
	err := c.Run()
	r.Exit = 0
	if err != nil {
		r.Exit = -1
		if ex, ok := err.(*exec.ExitError); ok {
			r.Exit = ex.ExitCode()
		}
	}
	f.Close()
	b, _ := os.ReadFile(path)
	if ctx.Err() != nil {
		r.Status = "REPLAY_BLOCKED"
	} else if interactive {
		r.Status = "DEV_EXITED"
	} else if r.Exit == 0 {
		r.Status = "STEP_PASSED_UNVERIFIED"
	} else if r.Exit == 125 || r.Exit == 126 || r.Exit == 127 || r.Exit < 0 {
		r.Status = "REPLAY_BLOCKED"
	} else {
		r.Evidence = failure.Compare(s.Lock.Failure.Log, string(b), s.Lock.Failure.Name, s.Lock.Failure.Command, r.Exit, true)
		r.Status = r.Evidence.Status
	}
	text := string(b)
	if strings.Contains(text, "Downloading ") || strings.Contains(text, "Downloaded ") {
		r.NetworkDownload = "OBSERVED"
	}
	if strings.Contains(text, "Installed ") {
		r.DependencySetup = true
	}
	if !interactive && r.Status != "REPLAY_BLOCKED" {
		s.DependencyHash = hash
		if e = Save(*s); e != nil {
			return r, e
		}
	}
	rb, _ := json.MarshalIndent(r, "", "  ")
	if e = lockfile.Atomic(filepath.Join(logs, r.Container+".json"), rb); e != nil {
		return r, e
	}
	if e = lockfile.Atomic(filepath.Join(s.Root, "last-debug.json"), rb); e != nil {
		return r, e
	}
	fmt.Fprintf(out, "Result: %s\nReport: %s\n", r.Status, filepath.Join(logs, r.Container+".json"))
	if r.Status == "REPLAY_BLOCKED" {
		return r, fmt.Errorf("debug execution blocked (exit %d)", r.Exit)
	}
	return r, nil
}
