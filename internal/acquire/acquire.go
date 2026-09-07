// Package acquire connects either evidence source to the one existing Resolver.
package acquire

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/failure"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/online"
	"github.com/huanglinfei091-cmd/runback/internal/resolver"
)

type Result struct {
	Lock                                     lockfile.Lock
	Evidence                                 github.RunEvidence
	Source, CacheStatus, CachePath, AuthMode string
}

func Resolve(ctx context.Context, url, bundle, job string, refresh bool) (Result, error) {
	var r Result
	if bundle != "" {
		b, e := github.ReadBundle(bundle)
		if e != nil {
			return r, e
		}
		l, e := resolver.Resolve(ctx, b, url, resolver.Options{Job: job})
		return Result{Lock: l, Evidence: *b, Source: "BUNDLE"}, e
	}
	for attempt := 0; attempt < 2; attempt++ {
		s, e := online.New(online.Options{Refresh: refresh || attempt > 0})
		if e != nil {
			return r, e
		}
		l, e := resolver.Resolve(ctx, s, url, resolver.Options{Job: job})
		e = s.Finish(e)
		r = Result{Lock: l, Evidence: s.Data, Source: "ONLINE", CacheStatus: s.CacheStatus, CachePath: s.CachePath, AuthMode: s.HTTP.Mode()}
		s.Close()
		if ce, ok := e.(*online.Error); ok && ce.Cause == "CACHE_INCOMPLETE" && attempt == 0 {
			continue
		}
		if e != nil {
			if _, ok := e.(*online.Error); !ok {
				e = &online.Error{Stage: "ACQUIRE", Cause: "EVIDENCE_UNAVAILABLE", Detail: fmt.Sprint(e)}
			}
		}
		return r, e
	}
	return r, fmt.Errorf("EVIDENCE_UNAVAILABLE: cache retry exhausted")
}
func Semantic(l lockfile.Lock) map[string]any {
	// Only acquisition-independent fields. The same historical Resolver establishes
	// actual checkout SHA, selected matrix, step, runtime and structured failure.
	return map[string]any{
		"repository": l.Repository, "run_id": l.RunID, "run_attempt": l.Attempt,
		"commit_sha": l.Commit, "event": l.Event, "workflow_path": l.WorkflowPath,
		"job_id": l.JobNumber, "workflow_job_id": l.JobID, "job_name": l.JobName,
		"runner": l.Runner, "matrix": l.Matrix, "runtime": l.ObservedVersions,
		"failed_step":      map[string]any{"name": l.Failure.Name, "command": l.Failure.Command, "shell": l.Failure.Shell, "working_directory": l.Failure.WorkingDirectory, "environment": l.Failure.Environment},
		"failure_evidence": failure.Parse(l.Failure.Log, l.Failure.Name, l.Failure.Command, nil),
	}
}
func SemanticHash(l lockfile.Lock) string {
	b, _ := json.Marshal(Semantic(l))
	return lockfile.Digest(string(b))
}
