//go:build linux

package session

import (
	"context"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicPublishNeverOverwritesExistingSession(t *testing.T) {
	base := t.TempDir()
	final := filepath.Join(base, "existing")
	temp := filepath.Join(base, ".tmp-candidate")
	os.Mkdir(final, 0700)
	os.Mkdir(temp, 0700)
	os.WriteFile(filepath.Join(temp, "state.json"), []byte("new"), 0600)
	if e := publishSession(temp, final); e == nil {
		t.Fatal("empty existing session overwritten")
	}
	if _, e := os.Stat(filepath.Join(final, "state.json")); !os.IsNotExist(e) {
		t.Fatal("collision published")
	}
	os.WriteFile(filepath.Join(final, "user-work"), []byte("preserve"), 0600)
	if e := publishSession(temp, final); e == nil {
		t.Fatal("existing session overwritten")
	}
	b, _ := os.ReadFile(filepath.Join(final, "user-work"))
	if string(b) != "preserve" {
		t.Fatal("old session damaged")
	}
	destination := filepath.Join(base, "new")
	if e := publishSession(temp, destination); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(temp); !os.IsNotExist(e) {
		t.Fatal("temp not renamed")
	}
}
func TestAtomicCreateAndFailedCreatePreserveActive(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	os.Mkdir(src, 0700)
	git(t, src, "init")
	put(t, filepath.Join(src, "main.py"), []byte("pass\n"))
	git(t, src, "add", ".")
	git(t, src, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "base")
	sha := git(t, src, "rev-parse", "HEAD")
	l := lockfile.Lock{Version: 1, URL: "https://github.com/acme/tool/actions/runs/123", Repository: "acme/tool", RunID: 123, Attempt: 1, JobNumber: 456, Commit: sha, HeadSHA: sha, JobID: "tests", Event: "push", EventJSON: []byte("{}"), Workflow: "name: CI\non: push\njobs:\n  tests:\n    runs-on: ubuntu-latest\n    steps:\n      - run: pytest\n"}
	base := filepath.Join(root, "sessions")
	p := replay.Plan{Root: root, Checkout: src, Home: root}
	r := runner.Result{Status: "SAME_FAILURE", Mode: "FULL_JOB"}
	s, e := Create(context.Background(), base, l, p, r)
	if e != nil {
		t.Fatal(e)
	}
	loaded, e := Load(base, "")
	if e != nil || loaded.ID != s.ID || strings.Contains(loaded.Worktree, ".tmp-") {
		t.Fatal(loaded, e)
	}
	for _, path := range []string{"original/.git", "worktree/.git", "cache", "state.json", "runback.lock", "reproduction.json"} {
		if _, e := os.Stat(filepath.Join(s.Root, path)); e != nil {
			t.Fatal(path, e)
		}
	}
	p.Checkout = filepath.Join(root, "missing")
	if _, e := Create(context.Background(), base, l, p, r); e == nil {
		t.Fatal("missing source accepted")
	}
	active, _ := Load(base, "")
	if active.ID != s.ID {
		t.Fatal("failed create changed active")
	}
	dirs, _ := os.ReadDir(base)
	for _, d := range dirs {
		if strings.HasPrefix(d.Name(), ".tmp-") {
			t.Fatal("partial session retained")
		}
	}
	for _, status := range []string{"LIKELY_MATCH", "PARTIAL_MATCH", "DIFFERENT_FAILURE", "REPLAY_BLOCKED"} {
		r.Status = status
		if _, e := Create(context.Background(), base, l, p, r); e == nil {
			t.Fatal(status)
		}
	}
}
