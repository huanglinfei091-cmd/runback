package main

import (
	"context"
	"github.com/huanglinfei091-cmd/runback/internal/acquire"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestURLCLIOverridePrecedenceAndUnsafePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RUNBACK_GITHUB_TOKEN", "cli-credential-must-not-appear")
	t.Setenv("RUNBACK_MAX_LOG_SIZE", "invalid-online-config-bundle-must-bypass")
	url := "https://github.com/pallets/flask/actions/runs/33397112701"
	bundle := "../../docs/m1/evidence/case-a-bundle.json"
	acquired, e := acquire.Resolve(context.Background(), url, bundle, "", false)
	if e != nil {
		t.Fatal(e)
	}
	l := acquired.Lock
	l.Image = "recorded-image"
	l.Network = "recorded-network"
	root := t.TempDir()
	lock := filepath.Join(root, "input.lock")
	if e = lockfile.Write(lock, l); e != nil {
		t.Fatal(e)
	}
	args := []string{url, "--bundle", bundle, "--lock", lock, "--work-dir", root, "--image", "cli-image", "--network", "cli-network", "--dry-run"}
	if code := mainRun(context.Background(), args); code != 0 {
		t.Fatal(code)
	}
	saved, e := lockfile.Read(lock)
	if e != nil || saved.Image != "cli-image" || saved.Network != "cli-network" {
		t.Fatal(saved.Image, saved.Network, e)
	}
	p, e := replay.Prepare(saved, root)
	if e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(p.Args, " ")
	if !strings.Contains(joined, "ubuntu-latest=cli-image") || !strings.Contains(joined, "--network cli-network") {
		t.Fatal(joined)
	}
	for _, path := range []string{lock, p.Workflow, p.Event, p.Empty} {
		b, _ := os.ReadFile(path)
		if strings.Contains(string(b), "cli-credential-must-not-appear") {
			t.Fatal("credential persisted", path)
		}
	}
	saved.Attempt++
	lockfile.Write(lock, saved)
	before, _ := os.ReadFile(lock)
	if code := mainRun(context.Background(), args); code != 2 {
		t.Fatal("mismatched lock executed", code)
	}
	after, _ := os.ReadFile(lock)
	if string(before) != string(after) {
		t.Fatal("mismatched lock overwritten")
	}
}

func TestBundleAndOnlineShareAutomaticAllocatorContract(t *testing.T) {
	for _, source := range []string{"ONLINE", "BUNDLE"} {
		root, e := acquire.AllocateWorkspace()
		if e != nil {
			t.Fatal(source, e)
		}
		t.Cleanup(func() { _ = os.RemoveAll(root) })
		if !strings.HasPrefix(root, "/tmp/rb/") || strings.Contains(root, "pallets") {
			t.Fatalf("%s allocator path %q", source, root)
		}
	}
}
