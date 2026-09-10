package doctor

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/huanglinfei091-cmd/runback/internal/online"
)

func testWorkspace(t *testing.T) func() (string, error) {
	t.Helper()
	return func() (string, error) {
		return os.MkdirTemp(t.TempDir(), "workspace-")
	}
}

func TestReadyDoctorKeepsCredentialsOutOfCommands(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "must-not-enter-command")
	t.Setenv("GH_TOKEN", "must-not-enter-command-either")
	var calls []string
	c := checker{
		lookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		run: func(_ context.Context, env []string, name string, args ...string) (string, error) {
			joined := strings.Join(append([]string{name}, args...), " ")
			calls = append(calls, joined)
			for _, item := range env {
				if strings.HasPrefix(item, "RUNBACK_GITHUB_TOKEN=") || strings.HasPrefix(item, "GH_TOKEN=") || strings.Contains(item, "must-not-enter-command") {
					t.Fatalf("credential entered subprocess: %q", item)
				}
			}
			if joined == "docker image inspect "+runnerImage+" --format {{.Id}}" {
				return "sha256:test", nil
			}
			return "test-version", nil
		},
		access: func(context.Context) (online.AccessStatus, error) {
			return online.AccessStatus{Mode: "token", Limit: 5000, Remaining: 4999}, nil
		},
		workspace: testWorkspace(t),
	}
	var out bytes.Buffer
	if !c.check(context.Background(), &out) {
		t.Fatal(out.String())
	}
	text := out.String()
	if !strings.Contains(text, "Ready to reproduce") || strings.Contains(text, "must-not-enter-command") {
		t.Fatal(text)
	}
	if strings.Contains(strings.Join(calls, "\n"), "network create") {
		t.Fatal(calls)
	}
}

func TestAnonymousQuotaExhaustionIsNotReady(t *testing.T) {
	c := checker{
		lookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		run: func(_ context.Context, _ []string, name string, args ...string) (string, error) {
			if name == "docker" && len(args) > 0 && args[0] == "image" {
				return "", errors.New("image absent")
			}
			return "test-version", nil
		},
		access: func(context.Context) (online.AccessStatus, error) {
			return online.AccessStatus{Mode: "anonymous", Limit: 60, Remaining: 0, Reset: "2030-03-17T17:46:40Z"}, nil
		},
		workspace: testWorkspace(t),
	}
	var out bytes.Buffer
	if c.check(context.Background(), &out) {
		t.Fatal("exhausted anonymous quota reported ready")
	}
	text := out.String()
	if !strings.Contains(text, "token not configured") || !strings.Contains(text, "quota exhausted") || !strings.Contains(text, "RunBack is not ready") {
		t.Fatal(text)
	}
}

func TestNetworkCheckOnlyReadsAndUsesExistingManagedFallback(t *testing.T) {
	var calls []string
	run := func(_ context.Context, _ []string, name string, args ...string) (string, error) {
		call := strings.Join(append([]string{name}, args...), " ")
		calls = append(calls, call)
		switch {
		case strings.HasPrefix(call, "docker image inspect"):
			return "sha256:test", nil
		case strings.Contains(call, "--network bridge"):
			return "", errors.New("default unavailable")
		case strings.HasPrefix(call, "docker network ls"):
			return "runback-existing\n", nil
		case strings.HasPrefix(call, "docker network inspect runback-existing"):
			return "bridge|false|true", nil
		case strings.Contains(call, "--network runback-existing"):
			return "", nil
		default:
			return "", errors.New("unexpected call")
		}
	}
	var out bytes.Buffer
	checkNetwork(context.Background(), &out, run, []string{"PATH=/bin"})
	all := strings.Join(calls, "\n")
	if !strings.Contains(out.String(), "managed fallback available") || strings.Contains(all, "network create") || strings.Contains(all, "subnet") || strings.Contains(all, "gateway") {
		t.Fatal(out.String(), all)
	}
}

func TestWorkspaceProbeRemovesOnlyAllocatedDirectory(t *testing.T) {
	parent := t.TempDir()
	keep := filepath.Join(parent, "keep")
	if err := os.WriteFile(keep, []byte("user"), 0600); err != nil {
		t.Fatal(err)
	}
	var root string
	err := checkWorkspace(func() (string, error) {
		var err error
		root, err = os.MkdirTemp(parent, "runback-owned-")
		return root, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("allocated directory remains", err)
	}
	if got, err := os.ReadFile(keep); err != nil || string(got) != "user" {
		t.Fatal("neighboring user data changed", err)
	}
}

func TestMissingActExplainsProblemCauseAndNextStep(t *testing.T) {
	c := checker{
		lookPath: func(name string) (string, error) {
			if name == "act" {
				return "", errors.New("missing")
			}
			return "/bin/" + name, nil
		},
		run: func(_ context.Context, _ []string, name string, args ...string) (string, error) {
			if name == "docker" && len(args) > 0 && args[0] == "image" {
				return "", errors.New("image absent")
			}
			return "test-version", nil
		},
		access: func(context.Context) (online.AccessStatus, error) {
			return online.AccessStatus{Mode: "token", Limit: 5000, Remaining: 4999}, nil
		},
		workspace: testWorkspace(t),
	}
	var out bytes.Buffer
	if c.check(context.Background(), &out) {
		t.Fatal("missing act reported ready")
	}
	text := out.String()
	for _, want := range []string{"Problem: act was not found on PATH.", "Cause: RunBack delegates", "Next: Install act"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestLowDiskSpaceIsActionableBlocker(t *testing.T) {
	c := checker{
		lookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		run: func(_ context.Context, _ []string, name string, args ...string) (string, error) {
			if name == "docker" && len(args) > 0 && args[0] == "image" {
				return "", errors.New("image absent")
			}
			return "test-version", nil
		},
		access: func(context.Context) (online.AccessStatus, error) {
			return online.AccessStatus{Mode: "token", Limit: 5000, Remaining: 4999}, nil
		},
		workspace: testWorkspace(t),
		diskFree:  func(string) (uint64, error) { return 1 << 30, nil },
		pathState: func() (bool, string) { return true, "/usr/local/bin" },
	}
	var out bytes.Buffer
	if c.check(context.Background(), &out) {
		t.Fatal("low disk space reported ready")
	}
	text := out.String()
	if !strings.Contains(text, "✗ Disk space (1.0 GiB free)") || !strings.Contains(text, "Next: df -h /tmp") {
		t.Fatal(text)
	}
}

func TestPathWarningDoesNotFailOtherwiseReadyDoctor(t *testing.T) {
	c := checker{
		lookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		run: func(_ context.Context, _ []string, name string, args ...string) (string, error) {
			if name == "docker" && len(args) > 0 && args[0] == "image" {
				return "", errors.New("image absent")
			}
			return "test-version", nil
		},
		access: func(context.Context) (online.AccessStatus, error) {
			return online.AccessStatus{Mode: "token", Limit: 5000, Remaining: 4999}, nil
		},
		workspace: testWorkspace(t),
		diskFree:  func(string) (uint64, error) { return 10 << 30, nil },
		pathState: func() (bool, string) { return false, "/tmp" },
	}
	var out bytes.Buffer
	if !c.check(context.Background(), &out) {
		t.Fatal(out.String())
	}
	if !strings.Contains(out.String(), "! RunBack is not on PATH") || !strings.Contains(out.String(), `export PATH="$HOME/.local/bin:$PATH"`) {
		t.Fatal(out.String())
	}
}
