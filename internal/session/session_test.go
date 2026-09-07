package session

import (
	"context"
	"encoding/json"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, e := run(context.Background(), dir, dir, "git", args...)
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func put(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func TestVerifySnapshotPreservesAllChanges(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "worktree")
	os.Mkdir(src, 0700)
	git(t, src, "init")
	for n, b := range map[string][]byte{"edit.txt": []byte("before\n"), "delete.txt": []byte("delete\n"), "rename.txt": []byte("rename\n"), "blob.bin": {0, 1, 2, 3}, "staged.txt": []byte("old\n"), ".gitignore": []byte("ignored.txt\n")} {
		put(t, filepath.Join(src, n), b)
	}
	git(t, src, "add", ".")
	git(t, src, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "base")
	sha := git(t, src, "rev-parse", "HEAD")
	dest := filepath.Join(root, "verify")
	git(t, root, "clone", src, dest)
	put(t, filepath.Join(src, "edit.txt"), []byte("after  \n"))
	os.Remove(filepath.Join(src, "delete.txt"))
	os.Rename(filepath.Join(src, "rename.txt"), filepath.Join(src, "new-name.txt"))
	put(t, filepath.Join(src, "blob.bin"), []byte{0, 255, 13, 8})
	put(t, filepath.Join(src, "staged.txt"), []byte("staged\n"))
	git(t, src, "add", "staged.txt")
	put(t, filepath.Join(src, "new file.txt"), []byte("untracked\n"))
	put(t, filepath.Join(src, "ignored.txt"), []byte("never copy\n"))
	put(t, filepath.Join(src, ".github/workflows/test.yaml"), []byte("changed workflow\n"))
	s := State{Root: root, Worktree: src, Lock: lockfile.Lock{Commit: sha}}
	c, e := Snapshot(context.Background(), s)
	if e != nil {
		t.Fatal(e)
	}
	if !c.WorkflowModified || len(c.Untracked) != 3 {
		t.Fatalf("%+v", c)
	}
	if e = Apply(context.Background(), s, c, dest, filepath.Join(root, "patch")); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"edit.txt", "blob.bin", "staged.txt", "new-name.txt", "new file.txt", ".github/workflows/test.yaml"} {
		a, _ := os.ReadFile(filepath.Join(src, name))
		b, e := os.ReadFile(filepath.Join(dest, name))
		if e != nil || string(a) != string(b) {
			t.Fatal(name, e)
		}
	}
	for _, name := range []string{"delete.txt", "rename.txt", "ignored.txt"} {
		if _, e = os.Stat(filepath.Join(dest, name)); !os.IsNotExist(e) {
			t.Fatal("must not exist", name)
		}
	}
}
func TestParentSymlinkCannotEscape(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if e := os.Symlink(other, filepath.Join(root, "link")); e != nil {
		t.Skip(e)
	}
	if _, e := safePath(root, "link/file"); e == nil {
		t.Fatal("must reject symlink parent")
	}
	for _, p := range []string{"../escape", "/etc/passwd"} {
		if _, e := safePath(root, p); e == nil {
			t.Fatal(p)
		}
	}
}
func TestContainersShareOneCheckoutAndPinnedImage(t *testing.T) {
	s := State{ID: "test", Root: "/sessions/test", Worktree: "/sessions/test/worktree", Cache: "/sessions/test/cache", ImageID: "sha256:abc", ImageOverride: "verified-image", PythonRoot: "/opt/python", UVPath: "/opt/uv/uv", Lock: lockfile.Lock{Failure: lockfile.Failure{WorkingDirectory: ".", Shell: "bash"}}}
	for _, interactive := range []bool{true, false} {
		args, e := DockerArgs(s, "container", interactive, "pytest")
		if e != nil {
			t.Fatal(e)
		}
		joined := strings.Join(args, " ")
		for _, want := range []string{"--rm", "src=/sessions/test/worktree,dst=/github/workspace", "sha256:abc", "src=/sessions/test/cache,dst=/runback/cache"} {
			if !strings.Contains(joined, want) {
				t.Fatal(joined)
			}
		}
		if strings.Contains(joined, "docker.sock") || strings.Contains(joined, "--privileged") {
			t.Fatal(joined)
		}
	}
}
func TestHostCredentialsNotInherited(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "must-not-leak")
	t.Setenv("GH_TOKEN", "must-not-leak")
	t.Setenv("RUNBACK_GITHUB_TOKEN", "must-not-leak")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "must-not-leak")
	t.Setenv("SSH_AUTH_SOCK", "must-not-leak")
	if strings.Contains(strings.Join(CleanEnv(t.TempDir()), "\n"), "must-not-leak") {
		t.Fatal("credential leak")
	}
}
func TestActiveSessionAndExplicitSelection(t *testing.T) {
	base := t.TempDir()
	l := lockfile.Lock{Version: 1, URL: "https://github.com/acme/tool/actions/runs/123", Repository: "acme/tool", RunID: 123, Attempt: 1, JobNumber: 456, Commit: strings.Repeat("a", 40), HeadSHA: strings.Repeat("a", 40), JobID: "test", Event: "push", EventJSON: json.RawMessage("{}")}
	for _, id := range []string{"one", "two"} {
		root := filepath.Join(base, id)
		s := State{Version: 1, ID: id, Root: root, Original: filepath.Join(root, "original"), Worktree: filepath.Join(root, "worktree"), Cache: filepath.Join(root, "cache"), Lock: l}
		if e := Save(s); e != nil {
			t.Fatal(e)
		}
		if e := Activate(s); e != nil {
			t.Fatal(e)
		}
	}
	for id, want := range map[string]string{"": "two", "one": "one"} {
		s, e := Load(base, id)
		if e != nil || s.ID != want {
			t.Fatal(id, s.ID, e)
		}
	}
	if _, e := Load(base, "../one"); e == nil {
		t.Fatal("invalid ID accepted")
	}
	s, _ := Load(base, "one")
	release, e := Acquire(s)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	if _, e = Acquire(s); e == nil {
		t.Fatal("concurrent mutation accepted")
	}
}
func TestManifestChangeInvalidatesDependencies(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init")
	put(t, filepath.Join(root, "pyproject.toml"), []byte("version = 1"))
	s := State{Root: root, Worktree: root}
	a, e := DependencyHash(context.Background(), s)
	if e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(root, "pyproject.toml"), []byte("version = 2"))
	b, e := DependencyHash(context.Background(), s)
	if e != nil || a == b {
		t.Fatal("manifest change ignored", e)
	}
	put(t, filepath.Join(root, "source.py"), []byte("pass"))
	c, e := DependencyHash(context.Background(), s)
	if e != nil || b != c {
		t.Fatal("source change invalidated tools", e)
	}
}
