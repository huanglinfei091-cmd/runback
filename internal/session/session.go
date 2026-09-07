// Package session owns the editable checkout and narrowly scoped debug tools.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/huanglinfei091-cmd/runback/internal/fingerprint"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"github.com/huanglinfei091-cmd/runback/internal/workflow"
)

type State struct {
	Version         int               `json:"version"`
	ID              string            `json:"id"`
	Root            string            `json:"root"`
	Original        string            `json:"original"`
	Worktree        string            `json:"worktree"`
	Cache           string            `json:"cache"`
	ImageOverride   string            `json:"image_override"`
	ImageID         string            `json:"image_id"`
	FullUVCachePath string            `json:"full_uv_cache_path,omitempty"`
	PythonVersion   string            `json:"python_version"`
	PythonRoot      string            `json:"python_root"`
	UVPath          string            `json:"uv_path"`
	Environment     map[string]string `json:"environment"`
	DebugBlocker    string            `json:"debug_blocker,omitempty"`
	DependencyHash  string            `json:"dependency_hash,omitempty"`
	Lock            lockfile.Lock     `json:"lock"`
}

var validID = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_-]{0,100}$")
var versionNumber = regexp.MustCompile("^[0-9]+[.][0-9]+[.][0-9]+$")
var envName = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_]*$")

func Base() (string, error) {
	h, e := os.UserHomeDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(h, ".runback", "sessions"), nil
}

// No inherited credentials are available to Git, act or Docker subprocesses.
func CleanEnv(home string) []string {
	out := []string{"HOME=" + home, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + filepath.Join(home, ".gitconfig"), "GIT_TERMINAL_PROMPT=0"}
	for _, k := range []string{"PATH", "DOCKER_HOST", "SYSTEMROOT", "WINDIR", "TMPDIR", "TEMP", "TMP"} {
		if v, ok := os.LookupEnv(k); ok {
			out = append(out, k+"="+v)
		}
	}
	return out
}
func output(ctx context.Context, home, dir, name string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	c.Env = CleanEnv(home)
	b, e := c.CombinedOutput()
	if e != nil {
		return string(b), fmt.Errorf("%s: %w: %.1200s", name, e, b)
	}
	return string(b), nil
}
func run(ctx context.Context, home, dir, name string, args ...string) (string, error) {
	b, e := output(ctx, home, dir, name, args...)
	return strings.TrimSpace(b), e
}
func Save(s State) error {
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	return lockfile.Atomic(filepath.Join(s.Root, "state.json"), b)
}
func Load(base, id string) (State, error) {
	var s State
	if id == "" {
		b, e := os.ReadFile(filepath.Join(base, "active"))
		if e != nil {
			return s, fmt.Errorf("no active session; reproduce a SAME_FAILURE first: %w", e)
		}
		id = strings.TrimSpace(string(b))
	}
	if !validID.MatchString(id) {
		return s, fmt.Errorf("invalid session ID")
	}
	root, e := filepath.Abs(filepath.Join(base, id))
	if e != nil {
		return s, e
	}
	b, e := os.ReadFile(filepath.Join(root, "state.json"))
	if e != nil {
		return s, e
	}
	if e = json.Unmarshal(b, &s); e != nil {
		return s, e
	}
	if s.Version != 1 || s.ID != id || s.Root != root || s.Worktree != filepath.Join(root, "worktree") || s.Original != filepath.Join(root, "original") || s.Cache != filepath.Join(root, "cache") {
		return s, fmt.Errorf("invalid session paths")
	}
	return s, s.Lock.Validate()
}
func Activate(s State) error {
	return lockfile.Atomic(filepath.Join(filepath.Dir(s.Root), "active"), []byte(s.ID+"\n"))
}

// Acquire serializes modifications to one session, including interactive dev use.
func Acquire(s State) (func(), error) {
	p := filepath.Join(s.Root, "busy")
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return nil, fmt.Errorf("session busy; if its process was killed, inspect %s before removing it", p)
	}
	fmt.Fprintf(f, "pid=%d\n", os.Getpid())
	f.Close()
	return func() { os.Remove(p) }, nil
}

// Create never reuses or overwrites an existing editable session.
func Create(ctx context.Context, base string, l lockfile.Lock, p replay.Plan, r runner.Result) (State, error) {
	var s State
	if r.Status != "SAME_FAILURE" || r.Mode != "FULL_JOB" || r.WorkspaceDirty {
		return s, fmt.Errorf("session needs a clean, verified SAME_FAILURE")
	}
	id := fmt.Sprintf("%d-%d-%d", l.RunID, l.JobNumber, time.Now().UnixNano())
	final, e := filepath.Abs(filepath.Join(base, id))
	if e != nil {
		return s, e
	}
	if e = os.MkdirAll(base, 0700); e != nil {
		return s, e
	}
	root, e := os.MkdirTemp(base, ".tmp-")
	if e != nil {
		return s, e
	}
	defer os.RemoveAll(root)
	s = State{Version: 1, ID: id, Root: root, Original: filepath.Join(root, "original"), Worktree: filepath.Join(root, "worktree"), Cache: filepath.Join(root, "cache"), ImageOverride: l.Image, ImageID: r.ImageID, Lock: l}
	if e = os.MkdirAll(s.Cache, 0700); e != nil {
		return s, e
	}
	for _, dest := range []string{s.Original, s.Worktree} {
		source := p.Checkout
		if dest == s.Worktree {
			source = s.Original
		}
		if _, e = run(ctx, p.Home, root, "git", "clone", "--no-hardlinks", "--no-checkout", source, dest); e != nil {
			return s, e
		}
		if _, e = run(ctx, p.Home, root, "git", "-C", dest, "-c", "core.hooksPath=/dev/null", "checkout", "--detach", l.Commit); e != nil {
			return s, e
		}
		// Keep the original public URL; no credentials or parent Git configuration.
		if _, e = run(ctx, p.Home, root, "git", "-C", dest, "remote", "set-url", "origin", "https://github.com/"+l.Repository+".git"); e != nil {
			return s, e
		}
	}
	if e = lockfile.Write(filepath.Join(root, "runback.lock"), l); e != nil {
		return s, e
	}
	rb, _ := json.MarshalIndent(r, "", "  ")
	if e = lockfile.Atomic(filepath.Join(root, "reproduction.json"), rb); e != nil {
		return s, e
	}
	if e = s.exportTools(ctx, p, r); e != nil {
		s.DebugBlocker = e.Error()
	}
	s.Root = final
	s.Original = filepath.Join(final, "original")
	s.Worktree = filepath.Join(final, "worktree")
	s.Cache = filepath.Join(final, "cache")
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return s, err
	}
	if e = lockfile.Atomic(filepath.Join(root, "state.json"), b); e != nil {
		return s, e
	}
	if e = l.Validate(); e != nil {
		return s, e
	}
	if e = publishSession(root, final); e != nil {
		return s, fmt.Errorf("SESSION_CREATE_FAILED: atomic publish/collision: %w", e)
	}
	return s, Activate(s)
}
func (s *State) exportTools(ctx context.Context, p replay.Plan, r runner.Result) error {
	w, e := workflow.Parse([]byte(s.Lock.Workflow))
	if e != nil {
		return e
	}
	job, ok := w.Jobs[s.Lock.JobID]
	if !ok {
		return fmt.Errorf("missing session job")
	}
	for i, step := range job.Steps {
		if i >= s.Lock.Failure.StepIndex {
			break
		}
		supported := false
		for _, prefix := range []string{"actions/checkout@", "actions/setup-python@", "astral-sh/setup-uv@"} {
			if strings.HasPrefix(step.Uses, prefix) {
				supported = true
			}
		}
		if !supported {
			return fmt.Errorf("DEBUG_UNSUPPORTED: earlier setup step %d cannot be carried into the narrow Python/uv debug environment", i+1)
		}
	}

	v := s.Lock.ObservedVersions["python-version"]
	if !versionNumber.MatchString(v) {
		return fmt.Errorf("DEBUG_UNSUPPORTED: M2 requires an observed CPython x.y.z")
	}
	if len(r.ContainerIDs) != 1 {
		return fmt.Errorf("DEBUG_UNSUPPORTED: expected one retained act container")
	}
	id := r.ContainerIDs[0]
	label, e := run(ctx, p.Home, p.Root, "docker", "inspect", id, "--format", "{{index .Config.Labels \"io.runback.case\"}}")
	if e != nil || label != s.Lock.Key() {
		return fmt.Errorf("refusing tool export from unrelated container")
	}
	s.PythonVersion = v
	s.PythonRoot = "/opt/hostedtoolcache/Python/" + v + "/x64"
	if _, e = run(ctx, p.Home, p.Root, "docker", "exec", id, "test", "-x", s.PythonRoot+"/bin/python3"); e != nil {
		return e
	}

	uvVersion := s.Lock.ObservedVersions["uv-version"]
	if !versionNumber.MatchString(uvVersion) {
		return fmt.Errorf("DEBUG_UNSUPPORTED: exact observed uv version missing")
	}
	uv := "/opt/hostedtoolcache/uv/" + uvVersion + "/x86_64/uv"
	if _, e = run(ctx, p.Home, p.Root, "docker", "exec", id, "test", "-x", uv); e != nil {
		return fmt.Errorf("observed uv tool unavailable: %w", e)
	}
	s.UVPath = uv
	for _, v := range []struct{ remote, local string }{{s.PythonRoot, filepath.Join(s.Cache, "python")}, {filepath.Dir(uv), filepath.Join(s.Cache, "uv-bin")}} {
		if _, e = run(ctx, p.Home, p.Root, "docker", "cp", id+":"+v.remote, v.local); e != nil {
			return e
		}
	}

	log, _ := os.ReadFile(filepath.Join(p.Root, "local.jsonl"))
	for _, line := range strings.Split(string(log), "\n") {
		var entry map[string]any
		if json.Unmarshal([]byte(line), &entry) == nil && entry["command"] == "set-env" && entry["name"] == "UV_CACHE_DIR" {
			path, _ := entry["arg"].(string)
			if strings.HasPrefix(path, "/root/.cache/") || strings.HasPrefix(path, "/tmp/") {
				s.FullUVCachePath = path
			}
		}
	}
	s.Environment = map[string]string{}
	// Expressions are taken from the observed remote failed-step env stanza, never guessed.
	for key, value := range s.Lock.Failure.Environment {
		if !envName.MatchString(key) {
			return fmt.Errorf("invalid environment key")
		}
		text := fmt.Sprint(value)
		if strings.Contains(text, "${{") {
			found := false
			for _, line := range strings.Split(s.Lock.Failure.Log, "\n") {
				line = fingerprint.Normalize(line)
				if strings.HasPrefix(line, key+": ") {
					text = strings.TrimPrefix(line, key+": ")
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("DEBUG_UNSUPPORTED: unresolved environment %s", key)
			}
		}
		s.Environment[key] = text
	}
	if strings.Contains(s.Lock.Failure.Command, "${{") {
		return fmt.Errorf("DEBUG_UNSUPPORTED: unresolved command expression")
	}
	if s.Lock.Failure.Shell != "bash" && s.Lock.Failure.Shell != "sh" {
		return fmt.Errorf("DEBUG_UNSUPPORTED: shell %s", s.Lock.Failure.Shell)
	}
	return nil
}

// StopReproductionContainers leaves retained evidence/shell containers on disk but
// no running process after Direct URL completion. It never touches another case.
func StopReproductionContainers(ctx context.Context, l lockfile.Lock, p replay.Plan, r runner.Result) error {
	for _, id := range r.ContainerIDs {
		label, e := run(ctx, p.Home, p.Root, "docker", "inspect", id, "--format", `{{index .Config.Labels "io.runback.case"}}`)
		if e != nil {
			return e
		}
		if label != l.Key() {
			return fmt.Errorf("container ownership mismatch")
		}
		if _, e = run(ctx, p.Home, p.Root, "docker", "stop", "--time", "2", id); e != nil {
			return e
		}
	}
	return nil
}
