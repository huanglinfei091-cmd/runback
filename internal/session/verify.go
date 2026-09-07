package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Changes struct {
	Tracked          []string `json:"tracked"`
	Untracked        []string `json:"untracked"`
	WorkflowModified bool     `json:"workflow_modified"`
	Patch            string   `json:"-"`
}

func splitNul(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, "\x00") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func Snapshot(ctx context.Context, s State) (Changes, error) {
	var c Changes
	// Compare to the frozen commit, including staged changes and local commits.
	raw, e := output(ctx, s.Root, s.Worktree, "git", "diff", "--binary", "--no-ext-diff", "--no-textconv", s.Lock.Commit, "--")
	if e != nil {
		return c, e
	}
	c.Patch = raw
	raw, e = output(ctx, s.Root, s.Worktree, "git", "diff", "--name-only", "-z", s.Lock.Commit, "--")
	if e != nil {
		return c, e
	}
	c.Tracked = splitNul(raw)
	raw, e = output(ctx, s.Root, s.Worktree, "git", "ls-files", "--others", "--exclude-standard", "-z")
	if e != nil {
		return c, e
	}
	c.Untracked = splitNul(raw)
	for _, p := range append(append([]string{}, c.Tracked...), c.Untracked...) {
		if strings.HasPrefix(p, ".github/workflows/") {
			c.WorkflowModified = true
		}
	}
	return c, nil
}
func safePath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(filepath.Clean(relative), ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid relative path %q", relative)
	}
	p := filepath.Join(root, relative)
	// Parent links must not redirect writes/reads outside the selected checkout.
	for parent := filepath.Dir(p); parent != root; parent = filepath.Dir(parent) {
		info, e := os.Lstat(parent)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink parent refused: %s", relative)
		}
	}
	return p, nil
}
func Apply(ctx context.Context, s State, c Changes, dest, patchFile string) error {
	if len(c.Tracked) > 0 {
		if e := lockfile.Atomic(patchFile, []byte(c.Patch)); e != nil {
			return e
		}
		if _, e := run(ctx, s.Root, dest, "git", "-c", "core.hooksPath=/dev/null", "apply", "--binary", "--index", patchFile); e != nil {
			return e
		}
	}
	for _, name := range c.Untracked {
		src, e := safePath(s.Worktree, name)
		if e != nil {
			return e
		}
		dst, e := safePath(dest, name)
		if e != nil {
			return e
		}
		info, e := os.Lstat(src)
		if e != nil {
			return e
		}
		if e = os.MkdirAll(filepath.Dir(dst), 0700); e != nil {
			return e
		}
		if _, e = os.Lstat(dst); e == nil {
			return fmt.Errorf("untracked destination already exists: %s", name)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, e := os.Readlink(src)
			if e != nil {
				return e
			}
			if filepath.IsAbs(link) || strings.HasPrefix(filepath.Clean(filepath.Join(filepath.Dir(name), link)), "..") {
				return fmt.Errorf("untracked symlink escapes checkout: %s", name)
			}
			if e = os.Symlink(link, dst); e != nil {
				return e
			}
		} else if info.Mode().IsRegular() {
			b, e := os.ReadFile(src)
			if e != nil {
				return e
			}
			if e = os.WriteFile(dst, b, info.Mode().Perm()); e != nil {
				return e
			}
		} else {
			return fmt.Errorf("unsupported untracked file type: %s", name)
		}
	}
	return nil
}

type VerifyResult struct {
	Status   string        `json:"status"`
	Workflow string        `json:"verification_workflow"`
	Changes  Changes       `json:"changes"`
	Result   runner.Result `json:"full_job"`
}

func Verify(ctx context.Context, s State, out io.Writer) (VerifyResult, error) {
	r := VerifyResult{Status: "VERIFY_BLOCKED", Workflow: "ORIGINAL_WORKFLOW"}
	release, e := Acquire(s)
	if e != nil {
		return r, e
	}
	defer release()
	root := filepath.Join(s.Root, "verifications", fmt.Sprint(time.Now().UnixNano()))
	if e = os.MkdirAll(root, 0700); e != nil {
		return r, e
	}
	r.Changes, e = Snapshot(ctx, s)
	if e != nil {
		return r, e
	}
	l := s.Lock
	l.Image = s.ImageID
	if r.Changes.WorkflowModified {
		fmt.Fprintln(out, "WORKFLOW_MODIFIED")
		// M2 makes a deliberate conservative choice: replay the frozen CI configuration.
		r.Workflow = "ORIGINAL_WORKFLOW_WITH_LOCAL_WORKFLOW_CHANGES_IGNORED"
		fmt.Fprintln(out, "Verification workflow: ORIGINAL_WORKFLOW (local workflow edits are not evaluated)")
	}
	p, e := replay.Prepare(l, root)
	if e != nil {
		return r, e
	}
	if e = cachedPlan(s, &p); e != nil {
		return r, e
	}
	if _, e = run(ctx, s.Root, root, "git", "clone", "--no-hardlinks", "--no-checkout", s.Original, p.Checkout); e != nil {
		return r, e
	}
	if _, e = run(ctx, s.Root, root, "git", "-C", p.Checkout, "-c", "core.hooksPath=/dev/null", "checkout", "--detach", l.Commit); e != nil {
		return r, e
	}
	if _, e = run(ctx, s.Root, root, "git", "-C", p.Checkout, "remote", "set-url", "origin", "https://github.com/"+l.Repository+".git"); e != nil {
		return r, e
	}
	if e = Apply(ctx, s, r.Changes, p.Checkout, filepath.Join(root, "changes.patch")); e != nil {
		return r, e
	}
	cb, _ := json.MarshalIndent(r.Changes, "", "  ")
	if e = lockfile.Atomic(filepath.Join(root, "changes.json"), cb); e != nil {
		return r, e
	}
	fmt.Fprintf(out, "Untracked files included: %v\nVerification workflow: %s\n", r.Changes.Untracked, r.Workflow)
	r.Result, e = runner.Execute(ctx, p, l, out)
	if e == nil {
		if r.Result.ExitCode == 0 && r.Result.TargetSucceeded {
			r.Status = "FULL_JOB_PASSED"
		} else if r.Result.Status == "SAME_FAILURE" {
			r.Status = "SAME_FAILURE"
		} else {
			r.Status = r.Result.Status
		}
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	if save := lockfile.Atomic(filepath.Join(root, "verify.json"), b); save != nil && e == nil {
		e = save
	}
	if save := lockfile.Atomic(filepath.Join(s.Root, "last-verify.json"), b); save != nil && e == nil {
		e = save
	}
	fmt.Fprintf(out, "Result: %s\nReport: %s\n", r.Status, filepath.Join(root, "verify.json"))
	return r, e
}
