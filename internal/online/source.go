package online

import (
	"context"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	Root    string
	Refresh bool
}
type Source struct {
	HTTP        *HTTP
	Root        string
	Refresh     bool
	Data        github.RunEvidence
	CacheStatus string
	CachePath   string
	temp        string
	pending     error
	jobID       int64
	raw         LogInfo
	target      github.Target
}

func New(o Options) (*Source, error) {
	h, e := NewHTTP()
	if e != nil {
		return nil, e
	}
	if o.Root == "" {
		home, e := os.UserHomeDir()
		if e != nil {
			return nil, e
		}
		o.Root = filepath.Join(home, ".runback", "cache", "github")
	}
	return &Source{HTTP: h, Root: o.Root, Refresh: o.Refresh}, nil
}
func (s *Source) Close() {
	if s.temp != "" {
		os.RemoveAll(s.temp)
		s.temp = ""
	}
}
func (s *Source) Run(ctx context.Context, t github.Target) (github.Run, error) {
	s.target = t
	var r github.Run
	path := fmt.Sprintf("%s/actions/runs/%d", t.Base(), t.RunID)
	cause := "RUN_METADATA_UNAVAILABLE"
	if t.Attempt > 0 {
		path += fmt.Sprintf("/attempts/%d", t.Attempt)
		cause = "RUN_ATTEMPT_UNAVAILABLE"
	}
	if e := s.HTTP.JSON(ctx, path, "ACQUIRE", cause, &r); e != nil {
		return r, e
	}
	if r.ID != t.RunID || !strings.EqualFold(r.Repository.FullName, t.Repository()) || r.RunAttempt < 1 || (t.Attempt > 0 && r.RunAttempt != t.Attempt) {
		return r, problem("ACQUIRE", "RUN_ATTEMPT_UNAVAILABLE", "run identity/attempt mismatch")
	}
	if r.Repository.Private {
		return r, problem("ACQUIRE", "RUN_METADATA_UNAVAILABLE", "private repositories are outside Direct URL MVP")
	}
	if r.Status != "completed" || r.Conclusion != "failure" {
		return r, problem("ACQUIRE", "RUN_METADATA_UNAVAILABLE", "run must be completed with conclusion=failure")
	}
	if r.Path == "" && r.WorkflowID > 0 {
		var workflow struct{ Path string }
		if e := s.HTTP.JSON(ctx, fmt.Sprintf("%s/actions/workflows/%d", t.Base(), r.WorkflowID), "WORKFLOW", "WORKFLOW_UNAVAILABLE", &workflow); e != nil {
			return r, e
		}
		r.Path = workflow.Path
	}
	p := strings.Split(r.Path, "@")[0]
	if !strings.HasPrefix(p, ".github/workflows/") || strings.Contains(p, "..") {
		return r, problem("WORKFLOW", "WORKFLOW_UNAVAILABLE", "workflow path unavailable from run/workflow metadata")
	}
	s.CachePath = filepath.Join(s.Root, t.Owner, t.Repo, fmt.Sprint(t.RunID), fmt.Sprintf("attempt-%d", r.RunAttempt))
	if !s.Refresh {
		data, manifest, e := s.load(r)
		if e == nil {
			s.Data = data
			s.jobID = manifest.JobID
			s.raw = LogInfo{manifest.JobLogSize, manifest.JobLogSHA}
			s.CacheStatus = "HIT"
			return s.Data.RunData, nil
		}
		if ce, ok := e.(*Error); ok {
			s.CacheStatus = ce.Cause
		} else {
			s.CacheStatus = "MISS"
		}
	} else {
		s.CacheStatus = "REFRESH"
	}
	tmpRoot := filepath.Join(s.Root, ".tmp")
	if e := os.MkdirAll(tmpRoot, 0700); e != nil {
		return r, e
	}
	CleanupTemps(tmpRoot)
	temp, e := os.MkdirTemp(tmpRoot, "runback-acquire-")
	if e != nil {
		return r, e
	}
	s.temp = temp
	s.Data = github.RunEvidence{URL: fmt.Sprintf("https://github.com/%s/actions/runs/%d/attempts/%d", t.Repository(), t.RunID, r.RunAttempt), RunData: r, Files: map[string]string{}, Logs: map[string]string{}}
	return r, nil
}
func (s *Source) Jobs(ctx context.Context, t github.Target, attempt int) ([]github.Job, error) {
	if attempt != s.Data.RunData.RunAttempt || t.RunID != s.target.RunID {
		return nil, problem("JOBS", "JOBS_UNAVAILABLE", "attempt mismatch")
	}
	if s.CacheStatus == "HIT" {
		return s.Data.JobData, nil
	}
	var jobs []github.Job
	seen := map[int64]bool{}
	for page := 1; page <= 10000; page++ {
		var response struct{ Jobs []github.Job }
		if e := s.HTTP.JSON(ctx, fmt.Sprintf("%s/actions/runs/%d/attempts/%d/jobs?per_page=100&page=%d", t.Base(), t.RunID, attempt, page), "JOBS", "JOBS_UNAVAILABLE", &response); e != nil {
			return nil, e
		}
		for _, j := range response.Jobs {
			if j.ID < 1 || seen[j.ID] || (j.RunID != 0 && j.RunID != t.RunID) || (j.RunAttempt != 0 && j.RunAttempt != attempt) {
				return nil, problem("JOBS", "JOBS_UNAVAILABLE", "job identity/attempt mismatch")
			}
			seen[j.ID] = true
			jobs = append(jobs, j)
		}
		if len(response.Jobs) < 100 {
			s.Data.JobData = jobs
			return jobs, nil
		}
	}
	return nil, problem("JOBS", "JOBS_UNAVAILABLE", "job pagination safety limit reached; incomplete jobs rejected")
}
func (s *Source) File(ctx context.Context, repo, path, sha string) ([]byte, error) {
	key := repo + "@" + sha + ":" + path
	if data, ok := s.Data.Files[key]; ok {
		return []byte(data), nil
	}
	if s.CacheStatus == "HIT" {
		s.pending = problem("CACHE", "CACHE_INCOMPLETE", "cached workflow snapshot missing")
		return nil, s.pending
	}
	if path != strings.Split(s.Data.RunData.Path, "@")[0] {
		return nil, problem("WORKFLOW", "WORKFLOW_UNAVAILABLE", "workflow path does not match metadata")
	}
	b, e := s.HTTP.File(ctx, repo, path, sha)
	if e == nil {
		s.Data.Files[key] = string(b)
	} else {
		// The historical Resolver records this as a replay blocker for bundles.
		// Online acquisition must retain the typed acquisition failure so
		// Finish cannot publish or execute incomplete evidence.
		s.pending = e
	}
	return b, e
}
func (s *Source) Get(ctx context.Context, path string) ([]byte, error) {
	if data, ok := s.Data.Logs[path]; ok {
		return []byte(data), nil
	}
	if s.CacheStatus == "HIT" {
		s.pending = problem("CACHE", "CACHE_INCOMPLETE", "cache does not contain the resolver-selected job")
		return nil, s.pending
	}
	prefix := s.target.Base() + "/actions/jobs/"
	idText := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/logs")
	id, e := strconv.ParseInt(idText, 10, 64)
	valid := false
	for _, j := range s.Data.JobData {
		if j.ID == id && j.Conclusion == "failure" {
			valid = true
		}
	}
	if e != nil || !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, "/logs") || !valid {
		s.pending = problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "job is not a failed job of this attempt")
		return nil, s.pending
	}
	info, e := s.HTTP.Log(ctx, path, s.temp)
	if e != nil {
		s.pending = e
		return nil, e
	}
	b, e := os.ReadFile(filepath.Join(s.temp, "job.raw.log"))
	if e != nil {
		s.pending = problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "raw evidence unavailable")
		return nil, s.pending
	}
	s.jobID = id
	s.raw = info
	s.Data.Logs[path] = string(b)
	return b, nil
}

// Finish promotes only complete evidence. Log failures swallowed by the historical
// resolver remain fatal acquisition errors here, without changing its bundle semantics.
func (s *Source) Finish(resolveErr error) error {
	if s.pending != nil {
		return s.pending
	}
	if resolveErr != nil {
		return resolveErr
	}
	if s.CacheStatus == "HIT" {
		return nil
	}
	return s.publish()
}
