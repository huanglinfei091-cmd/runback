package main

import (
	"context"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"github.com/huanglinfei091-cmd/runback/internal/session"
	"io"
	"path/filepath"
)

type sessionCreator func(context.Context, string, lockfile.Lock, replay.Plan, runner.Result) (session.State, error)
type containerStopper func(context.Context, lockfile.Lock, replay.Plan, runner.Result) error

// Reproduction is already persisted and reported before this optional session step.
func completeReproduction(ctx context.Context, out io.Writer, base string, l lockfile.Lock, p replay.Plan, r runner.Result, create sessionCreator, stop containerStopper) int {
	if r.Status != "SAME_FAILURE" {
		return 1
	}
	fmt.Fprintln(out, "✓ SAME_FAILURE reproduced")
	if r.Mode != "FULL_JOB" || r.WorkspaceDirty {
		return 0
	}
	s, e := create(ctx, base, l, p, r)
	if e != nil {
		fmt.Fprintf(out, "SESSION_CREATE_FAILED: %v\nReproduction evidence: %s\n", e, filepath.Join(p.Root, "evidence.json"))
	} else {
		fmt.Fprintf(out, "Session created: %s\nWorkspace: %s\nNext:\n  runback dev\n", s.ID, s.Worktree)
	}
	if ce := stop(ctx, l, p, r); ce != nil {
		fmt.Fprintln(out, "CONTAINER_CLEANUP_FAILED:", ce)
	} else if e == nil {
		fmt.Fprintln(out, "No container remains running. Session files remain on disk.")
	}
	if e == nil && s.DebugBlocker != "" {
		fmt.Fprintln(out, s.DebugBlocker)
	}
	return 0
}
