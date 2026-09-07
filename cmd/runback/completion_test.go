package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"github.com/huanglinfei091-cmd/runback/internal/session"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionSessionGatingAndFailure(t *testing.T) {
	for _, tc := range []struct {
		status, mode  string
		fail, created bool
		exit          int
	}{
		{"SAME_FAILURE", "FULL_JOB", false, true, 0},
		{"SAME_FAILURE", "FULL_JOB", true, true, 0},
		{"SAME_FAILURE", "STEP_ONLY", false, false, 0},
		{"LIKELY_MATCH", "FULL_JOB", false, false, 1},
		{"PARTIAL_MATCH", "FULL_JOB", false, false, 1},
		{"DIFFERENT_FAILURE", "FULL_JOB", false, false, 1},
		{"REPLAY_BLOCKED", "FULL_JOB", false, false, 1},
	} {
		t.Run(tc.status+tc.mode+string(rune('0'+tc.exit)), func(t *testing.T) {
			root := t.TempDir()
			evidence := filepath.Join(root, "evidence.json")
			os.WriteFile(evidence, []byte("independent evidence"), 0600)
			called := false
			create := func(context.Context, string, lockfile.Lock, replay.Plan, runner.Result) (session.State, error) {
				called = true
				if tc.fail {
					return session.State{}, errors.New("disk unavailable")
				}
				return session.State{ID: "created", Worktree: root}, nil
			}
			stop := func(context.Context, lockfile.Lock, replay.Plan, runner.Result) error { return nil }
			var out bytes.Buffer
			code := completeReproduction(context.Background(), &out, root, lockfile.Lock{}, replay.Plan{Root: root}, runner.Result{Status: tc.status, Mode: tc.mode}, create, stop)
			if code != tc.exit || called != tc.created {
				t.Fatal(code, called, out.String())
			}
			b, _ := os.ReadFile(evidence)
			if string(b) != "independent evidence" {
				t.Fatal("evidence lost")
			}
			if tc.fail && (!strings.Contains(out.String(), "SESSION_CREATE_FAILED") || !strings.Contains(out.String(), "SAME_FAILURE reproduced") || !strings.Contains(out.String(), evidence) || strings.Contains(out.String(), "Session created:")) {
				t.Fatal(out.String())
			}
		})
	}
}
