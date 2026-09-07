package acquire

import (
	"context"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resolved(t *testing.T) Result {
	t.Helper()
	r, e := Resolve(context.Background(), "https://github.com/pallets/flask/actions/runs/33397112701", "../../docs/m1/evidence/case-a-bundle.json", "", true)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestExplicitBundleBypassesOnlineEvenInvalidOnlineConfig(t *testing.T) {
	t.Setenv("RUNBACK_MAX_LOG_SIZE", "invalid")
	t.Setenv("RUNBACK_GITHUB_TOKEN", "never-use-this-token")
	r := resolved(t)
	if r.Source != "BUNDLE" || r.CachePath != "" || r.AuthMode != "" {
		t.Fatal(r.Source)
	}
}
func TestSemanticIdentityIncludesAttemptIgnoresExecutionOverrides(t *testing.T) {
	l := resolved(t).Lock
	same := l
	same.Image = "explicit-image"
	same.Network = "explicit-network"
	same.Warnings = append([]string{}, "acquisition timestamp differs")
	if SemanticHash(l) != SemanticHash(same) {
		t.Fatal("execution/cache metadata changed canonical semantics")
	}
	same.Attempt++
	if SemanticHash(l) == SemanticHash(same) {
		t.Fatal("attempt excluded from semantics")
	}
}
func TestExplicitLockIdentitySafety(t *testing.T) {
	l := resolved(t).Lock
	for _, field := range []string{"repository", "run", "attempt", "commit", "job", "workflow-job"} {
		t.Run(field, func(t *testing.T) {
			wrong := l
			switch field {
			case "repository":
				wrong.Repository = "another/project"
			case "run":
				wrong.RunID++
			case "attempt":
				wrong.Attempt++
			case "commit":
				wrong.Commit = strings.Repeat("a", 40)
			case "job":
				wrong.JobNumber++
			case "workflow-job":
				wrong.JobID = "different"
			}
			p := filepath.Join(t.TempDir(), "runback.lock")
			if e := lockfile.Write(p, wrong); e != nil {
				t.Fatal(e)
			}
			before, _ := os.ReadFile(p)
			if _, e := ExistingLock(p, l); e == nil || !strings.Contains(e.Error(), "LOCK_IDENTITY_MISMATCH") {
				t.Fatal(e)
			}
			after, _ := os.ReadFile(p)
			if string(before) != string(after) {
				t.Fatal("old lock overwritten")
			}
		})
	}
	p := filepath.Join(t.TempDir(), "lock")
	l.Image = "recorded-image"
	lockfile.Write(p, l)
	old, e := ExistingLock(p, l)
	if e != nil || old.Image != "recorded-image" {
		t.Fatal(e)
	}
}
func TestWorkdirProtectsUserDataAndSymlink(t *testing.T) {
	l := resolved(t).Lock
	base := t.TempDir()
	root := filepath.Join(base, l.Key())
	os.MkdirAll(root, 0700)
	file := filepath.Join(root, "user-source.py")
	os.WriteFile(file, []byte("valuable user changes"), 0600)
	if e := SafeWorkdir(base, l); e == nil || !strings.Contains(e.Error(), "WORKDIR_NOT_SAFE") {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(file)
	if string(b) != "valuable user changes" {
		t.Fatal("user file damaged")
	}
	safe := t.TempDir()
	if e := SafeWorkdir(safe, l); e != nil {
		t.Fatal(e)
	}
	if e := SafeWorkdir(safe, l); e != nil {
		t.Fatal("managed path rejected", e)
	}
	wrong := l
	wrong.Commit = strings.Repeat("b", 40)
	if e := SafeWorkdir(safe, wrong); e == nil {
		t.Fatal("foreign identity accepted")
	}
	link := filepath.Join(t.TempDir(), "link")
	if e := os.Symlink(base, link); e != nil {
		t.Skip(e)
	}
	if e := SafeWorkdir(link, l); e == nil {
		t.Fatal("symlink accepted")
	}
}
