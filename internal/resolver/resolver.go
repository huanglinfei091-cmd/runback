package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/fingerprint"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/workflow"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Source interface {
	Run(context.Context, github.Target) (github.Run, error)
	Jobs(context.Context, github.Target, int) ([]github.Job, error)
	File(context.Context, string, string, string) ([]byte, error)
	Get(context.Context, string) ([]byte, error)
}
type Options struct{ Job string }

func Resolve(ctx context.Context, c Source, raw string, o Options) (lockfile.Lock, error) {
	var l lockfile.Lock
	t, e := github.ParseURL(raw)
	if e != nil {
		return l, e
	}
	r, e := c.Run(ctx, t)
	if e != nil {
		return l, e
	}
	if r.Repository.Private {
		return l, fmt.Errorf("OUT_OF_SCOPE: private repository")
	}
	if r.ID != t.RunID || !strings.EqualFold(r.Repository.FullName, t.Repository()) {
		return l, fmt.Errorf("run identity mismatch")
	}
	if r.Status != "completed" || r.Conclusion != "failure" {
		return l, fmt.Errorf("expected a completed failed run, got %s/%s", r.Status, r.Conclusion)
	}
	if r.Event != "push" && r.Event != "pull_request" {
		return l, fmt.Errorf("OUT_OF_SCOPE: event %s", r.Event)
	}
	jobs, e := c.Jobs(ctx, t, r.RunAttempt)
	if e != nil {
		return l, e
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].ID < jobs[j].ID })
	path := strings.Split(r.Path, "@")[0]
	if !strings.HasPrefix(path, ".github/workflows/") || strings.Contains(path, "..") {
		return l, fmt.Errorf("WORKFLOW_UNSUPPORTED: non-local workflow path")
	}
	sourceRepo := r.Repository.FullName
	content, e := c.File(ctx, sourceRepo, path, r.HeadSHA)
	if e != nil && r.HeadRepository.FullName != "" {
		content, e = c.File(ctx, r.HeadRepository.FullName, path, r.HeadSHA)
	}
	if e != nil {
		return l, e
	}
	w, e := workflow.Parse(content)
	if e != nil {
		return l, e
	}
	var chosen github.Job
	var sel workflow.Selection
	var reasons []string
	eligible := 0
	for _, j := range jobs {
		if j.Conclusion != "failure" {
			continue
		}
		if o.Job != "" && o.Job != j.Name && o.Job != fmt.Sprint(j.ID) {
			continue
		}
		s, err := workflow.Select(w, j.Name)
		if err != nil {
			reasons = append(reasons, err.Error())
			continue
		}
		if s.Runner != "ubuntu-latest" || contains(j.Labels, "self-hosted") {
			reasons = append(reasons, j.Name+": RUNNER_UNSUPPORTED")
			continue
		}
		eligible++
		if chosen.ID == 0 {
			chosen = j
			sel = s
		}
	}
	if chosen.ID == 0 {
		return l, fmt.Errorf("no uniquely resolvable supported failed job: %s", strings.Join(reasons, "; "))
	}
	var failed github.Step
	for _, s := range chosen.Steps {
		if s.Conclusion == "failure" {
			failed = s
			break
		}
	}
	if failed.Number == 0 {
		return l, fmt.Errorf("failure has no failed step")
	}
	command, index, e := workflow.FailedCommand(sel.Job, failed.Name, sel.Matrix)
	if e != nil {
		return l, e
	}
	l = lockfile.Lock{Version: 1, URL: raw, Repository: r.Repository.FullName, RunID: r.ID, Attempt: r.RunAttempt, HeadSHA: r.HeadSHA, Commit: r.HeadSHA, CheckoutSource: "run.head_sha", Branch: r.HeadBranch, Event: r.Event, Actor: r.Actor.Login, WorkflowPath: path, WorkflowRef: r.Path, Workflow: string(content), JobID: sel.ID, JobName: chosen.Name, JobNumber: chosen.ID, Matrix: sel.Matrix, Runner: sel.Runner, Image: "ghcr.io/catthehacker/ubuntu:act-latest", ToolVersions: map[string]string{}, Failure: lockfile.Failure{Name: failed.Name, Command: command, StepIndex: index}, Warnings: []string{}, Blockers: []string{}}
	if eligible > 1 {
		l.Warnings = append(l.Warnings, fmt.Sprintf("%d supported failed jobs; selected lowest job ID automatically; --job chooses another", eligible))
	}
	logs, e := c.Get(ctx, fmt.Sprintf("%s/actions/jobs/%d/logs", t.Base(), chosen.ID))
	if e != nil {
		l.Warnings = append(l.Warnings, e.Error())
	} else {
		l.Failure.Log = fingerprint.StepLog(string(logs), failed.StartedAt, failed.CompletedAt)
	}
	l.ObservedVersions = ObservedVersions(string(logs))
	checkoutSHA := checkoutCommit(string(logs))
	if checkoutSHA != "" {
		l.Commit = checkoutSHA
		l.CheckoutSource = "checkout git log in original job log"
	}
	if r.Event == "pull_request" && checkoutSHA == "" {
		l.Blockers = append(l.Blockers, "ENVIRONMENT_MISMATCH: historical PR checkout SHA unavailable; refusing to substitute head for merge commit")
	}
	if l.Commit != r.HeadSHA {
		// The executed workflow is read at the historical merge commit, never the current branch.
		b, err := c.File(ctx, sourceRepo, path, l.Commit)
		if err != nil {
			l.Blockers = append(l.Blockers, "historical checkout workflow unavailable")
		} else {
			w2, err := workflow.Parse(b)
			if err != nil {
				return l, err
			}
			s2, err := workflow.Select(w2, chosen.Name)
			if err != nil {
				return l, err
			}
			cmd, idx, err := workflow.FailedCommand(s2.Job, failed.Name, s2.Matrix)
			if err != nil {
				return l, err
			}
			w = w2
			sel = s2
			l.Workflow = string(b)
			l.Matrix = s2.Matrix
			l.JobID = s2.ID
			l.Runner = s2.Runner
			l.Failure.Command = cmd
			l.Failure.StepIndex = idx
		}
	}
	if l.Runner != "ubuntu-latest" {
		l.Blockers = append(l.Blockers, "RUNNER_UNSUPPORTED")
	}
	if sel.Job.Needs != nil {
		l.Blockers = append(l.Blockers, "WORKFLOW_UNSUPPORTED: needs outputs/dependencies are not reconstructed")
	}
	if sel.Job.Uses != "" || sel.Job.Container != nil || sel.Job.Services != nil {
		l.Blockers = append(l.Blockers, "WORKFLOW_UNSUPPORTED: reusable workflow, job container or services")
	}
	if sel.Job.Environment != nil {
		l.Blockers = append(l.Blockers, "SECRETS_REQUIRED: repository environment is not reconstructible")
	}
	relevant, _ := workflow.Replay(w, sel)
	if regexp.MustCompile(`\b(?:secrets|vars)\s*(?:\.|\[)`).Match(relevant) {
		l.Blockers = append(l.Blockers, "SECRETS_REQUIRED: workflow references secrets or repository variables")
	}
	checkouts := 0
	for _, s := range sel.Job.Steps {
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			checkouts++
			for _, k := range []string{"repository", "ref", "path", "submodules", "lfs"} {
				if v, ok := s.With[k]; ok && fmt.Sprint(v) != "false" {
					l.Blockers = append(l.Blockers, "WORKFLOW_UNSUPPORTED: custom checkout "+k)
				}
			}
		}
		for _, k := range []string{"node-version", "python-version", "go-version", "node-version-file", "python-version-file", "go-version-file"} {
			if v, ok := s.With[k]; ok {
				version, known := workflow.Render(fmt.Sprint(v), sel.Matrix)
				if known {
					l.ToolVersions[k] = version
				} else {
					l.Warnings = append(l.Warnings, "unresolved tool version: "+k)
				}
			}
		}
	}
	if checkouts != 1 {
		l.Blockers = append(l.Blockers, "WORKFLOW_UNSUPPORTED: V1 requires exactly one standard actions/checkout")
	}
	execution := workflow.Execution(w, l.JobID, l.Failure.StepIndex)
	l.Failure.Shell = execution.Shell
	l.Failure.WorkingDirectory = execution.WorkingDirectory
	l.Failure.Environment = execution.Environment
	prNumber := 0
	if m := regexp.MustCompile(`refs/(?:remotes/)?pull/(\d+)/merge`).FindStringSubmatch(string(logs)); m != nil {
		prNumber, _ = strconv.Atoi(m[1])
	}
	l.EventJSON = event(r, l.Commit, prNumber)
	if len(l.Failure.Log) == 0 {
		l.Warnings = append(l.Warnings, "remote failed-step log unavailable; fingerprint result will be unverified")
	}
	l.Warnings = append(l.Warnings, "event payload is reconstructed; original secrets, caches, artifacts, runner VM and floating action/tool versions are not recoverable", "act uses a container approximation; the original job conditions and remaining steps are preserved")
	return l, l.Validate()
}
func contains(values []string, v string) bool {
	for _, s := range values {
		if s == v {
			return true
		}
	}
	return false
}
func checkoutCommit(log string) string {
	want := false
	for _, line := range strings.Split(log, "\n") {
		s := fingerprint.Normalize(line)
		if want && lockfile.SHA.MatchString(s) {
			return s
		}
		want = strings.Contains(s, "git log -1 --format=%H") || strings.Contains(s, "git log -1 --format='%H'")
	}
	return ""
}
func event(r github.Run, commit string, prNumber int) json.RawMessage {
	repo := map[string]any{"full_name": r.Repository.FullName, "default_branch": r.Repository.DefaultBranch, "clone_url": "https://github.com/" + r.Repository.FullName + ".git", "name": strings.Split(r.Repository.FullName, "/")[1], "owner": map[string]any{"login": strings.Split(r.Repository.FullName, "/")[0]}}
	payload := map[string]any{"repository": repo, "after": commit, "ref": "refs/heads/" + r.HeadBranch, "sender": map[string]string{"login": r.Actor.Login}}
	if r.Event == "pull_request" && len(r.PullRequests) > 0 {
		pr := r.PullRequests[0]
		payload["number"] = pr.Number
		payload["action"] = "synchronize"
		payload["pull_request"] = pr
		payload["ref"] = fmt.Sprintf("refs/pull/%d/merge", pr.Number)
	}
	if r.Event == "pull_request" && len(r.PullRequests) == 0 && prNumber > 0 {
		payload["number"] = prNumber
		payload["action"] = "synchronize"
		payload["ref"] = fmt.Sprintf("refs/pull/%d/merge", prNumber)
		payload["pull_request"] = map[string]any{"number": prNumber, "merge_commit_sha": commit, "head": map[string]any{"sha": r.HeadSHA, "ref": r.HeadBranch}, "base": map[string]any{"repo": repo}}
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return b
}
