package online

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	Schema       int    `json:"schema_version"`
	Repository   string `json:"repository"`
	RunID        int64  `json:"run_id"`
	Attempt      int    `json:"run_attempt"`
	JobID        int64  `json:"job_id"`
	HeadSHA      string `json:"head_sha"`
	WorkflowPath string `json:"workflow_path"`
	WorkflowSHA  string `json:"workflow_sha256"`
	JobsSHA      string `json:"jobs_sha256"`
	RunSHA       string `json:"run_sha256"`
	FilesSHA     string `json:"files_sha256"`
	JobLogSHA    string `json:"job_log_sha256"`
	JobLogSize   int64  `json:"job_log_size"`
	FetchedAt    string `json:"fetched_at"`
}

func digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func fileDigest(path string) (string, int64, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", 0, e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil)), n, e
}
func CleanupTemps(root string) {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "runback-acquire-") || len(e.Name()) <= len("runback-acquire-") {
			continue
		}
		info, err := e.Info()
		if err == nil && time.Since(info.ModTime()) > 24*time.Hour {
			_ = os.RemoveAll(filepath.Join(root, e.Name()))
		}
	}
}
func (s *Source) headWorkflow() (string, bool) {
	r := s.Data.RunData
	path := strings.Split(r.Path, "@")[0]
	for _, repo := range []string{r.Repository.FullName, r.HeadRepository.FullName} {
		if b, ok := s.Data.Files[repo+"@"+r.HeadSHA+":"+path]; ok {
			return b, true
		}
	}
	return "", false
}
func (s *Source) load(expected github.Run) (github.RunEvidence, Manifest, error) {
	var data github.RunEvidence
	var m Manifest
	miss := func(cause string) (github.RunEvidence, Manifest, error) {
		return data, m, problem("CACHE", cause, "cache rejected; acquiring fresh evidence")
	}
	resolved, e := filepath.EvalSymlinks(s.CachePath)
	if e != nil {
		return miss("CACHE_INCOMPLETE")
	}
	entries, _ := filepath.Abs(filepath.Join(s.Root, ".entries"))
	if !strings.HasPrefix(resolved, entries+string(os.PathSeparator)) {
		return miss("CACHE_CORRUPT")
	}
	return s.validateDirectory(resolved, expected)
}

// validateDirectory is shared by cache reads and pre-publication validation.
func (s *Source) validateDirectory(resolved string, expected github.Run) (github.RunEvidence, Manifest, error) {
	var data github.RunEvidence
	var m Manifest
	miss := func(cause string) (github.RunEvidence, Manifest, error) {
		return data, m, problem("CACHE", cause, "cache rejected; acquiring fresh evidence")
	}
	b, e := os.ReadFile(filepath.Join(resolved, "manifest.json"))
	if e != nil {
		return miss("CACHE_INCOMPLETE")
	}
	if json.Unmarshal(b, &m) != nil || m.Schema != 1 || m.Repository != expected.Repository.FullName || m.RunID != expected.ID || m.Attempt != expected.RunAttempt || m.HeadSHA != expected.HeadSHA || m.WorkflowPath != strings.Split(expected.Path, "@")[0] {
		return miss("CACHE_CORRUPT")
	}
	if m.JobLogSize <= 0 || m.JobLogSize > s.HTTP.MaxLog {
		return miss("CACHE_CORRUPT")
	}
	checks := map[string]string{"run.json": m.RunSHA, "jobs.json": m.JobsSHA, "files.json": m.FilesSHA, "workflow.yml": m.WorkflowSHA, "job.raw.log": m.JobLogSHA}
	blobs := map[string][]byte{}
	for name, want := range checks {
		p := filepath.Join(resolved, name)
		info, e := os.Lstat(p)
		if e != nil {
			return miss("CACHE_INCOMPLETE")
		}
		if !info.Mode().IsRegular() {
			return miss("CACHE_CORRUPT")
		}
		sha, n, e := fileDigest(p)
		if e != nil || sha != want || (name == "job.raw.log" && n != m.JobLogSize) {
			return miss("CACHE_CORRUPT")
		}
		raw, e := os.ReadFile(p)
		if e != nil || s.HTTP.containsToken(raw) {
			return miss("CACHE_CORRUPT")
		}
		blobs[name] = raw
	}
	if json.Unmarshal(blobs["run.json"], &data.RunData) != nil || json.Unmarshal(blobs["jobs.json"], &data.JobData) != nil || json.Unmarshal(blobs["files.json"], &data.Files) != nil {
		return miss("CACHE_CORRUPT")
	}
	r := data.RunData
	if r.ID != m.RunID || r.RunAttempt != m.Attempt || r.HeadSHA != m.HeadSHA || r.Repository.FullName != m.Repository || r.Repository.Private || r.Status != "completed" || r.Conclusion != "failure" {
		return miss("CACHE_CORRUPT")
	}
	found := false
	seen := map[int64]bool{}
	for _, j := range data.JobData {
		if j.ID <= 0 || seen[j.ID] || (j.RunID != 0 && j.RunID != m.RunID) || (j.RunAttempt != 0 && j.RunAttempt != m.Attempt) {
			return miss("CACHE_CORRUPT")
		}
		seen[j.ID] = true
		if j.ID == m.JobID && j.Conclusion == "failure" {
			found = true
		}
	}
	if !found {
		return miss("CACHE_CORRUPT")
	}
	head, ok := data.Files[r.Repository.FullName+"@"+r.HeadSHA+":"+m.WorkflowPath]
	if !ok {
		head, ok = data.Files[r.HeadRepository.FullName+"@"+r.HeadSHA+":"+m.WorkflowPath]
	}
	if !ok || head != string(blobs["workflow.yml"]) {
		return miss("CACHE_CORRUPT")
	}
	data.URL = fmt.Sprintf("https://github.com/%s/actions/runs/%d/attempts/%d", m.Repository, m.RunID, m.Attempt)
	data.Logs = map[string]string{fmt.Sprintf("%s/actions/jobs/%d/logs", s.target.Base(), m.JobID): string(blobs["job.raw.log"])}
	return data, m, nil
}
func (s *Source) publish() error {
	w, ok := s.headWorkflow()
	if !ok {
		return problem("WORKFLOW", "WORKFLOW_UNAVAILABLE", "head_sha workflow snapshot missing")
	}
	if s.jobID <= 0 || s.raw.Size <= 0 {
		return problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "selected raw job evidence missing")
	}
	r := s.Data.RunData
	m := Manifest{Schema: 1, Repository: r.Repository.FullName, RunID: r.ID, Attempt: r.RunAttempt, JobID: s.jobID, HeadSHA: r.HeadSHA, WorkflowPath: strings.Split(r.Path, "@")[0], JobLogSHA: s.raw.SHA256, JobLogSize: s.raw.Size, FetchedAt: time.Now().UTC().Format(time.RFC3339)}
	objects := map[string]any{"run.json": r, "jobs.json": s.Data.JobData, "files.json": s.Data.Files}
	shas := map[string]string{}
	for name, v := range objects {
		b, e := json.Marshal(v)
		if e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(s.temp, name), b, 0600); e != nil {
			return e
		}
		shas[name] = digest(b)
	}
	m.RunSHA = shas["run.json"]
	m.JobsSHA = shas["jobs.json"]
	m.FilesSHA = shas["files.json"]
	m.WorkflowSHA = digest([]byte(w))
	if e := os.WriteFile(filepath.Join(s.temp, "workflow.yml"), []byte(w), 0600); e != nil {
		return e
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if e := os.WriteFile(filepath.Join(s.temp, "manifest.json"), b, 0600); e != nil {
		return e
	}
	sha, n, e := fileDigest(filepath.Join(s.temp, "job.raw.log"))
	if e != nil || sha != m.JobLogSHA || n != m.JobLogSize {
		return problem("CACHE", "CACHE_CORRUPT", "raw evidence changed before publish")
	}
	if _, _, e = s.validateDirectory(s.temp, r); e != nil {
		return e
	}
	// Immutable complete generations + atomic symlink swap also make refresh atomic.
	// Readers see the previous complete generation or the next one, never partial files.
	entries := filepath.Join(s.Root, ".entries")
	if e = os.MkdirAll(entries, 0700); e != nil {
		return e
	}
	generation := filepath.Join(entries, filepath.Base(s.temp))
	if _, e = os.Lstat(generation); !os.IsNotExist(e) {
		return problem("CACHE", "CACHE_CORRUPT", "cache generation collision")
	}
	if e = os.Rename(s.temp, generation); e != nil {
		return e
	}
	s.temp = ""
	parent := filepath.Dir(s.CachePath)
	if e = os.MkdirAll(parent, 0700); e != nil {
		return e
	}
	link := filepath.Join(parent, ".publish-"+filepath.Base(generation))
	if e = os.Symlink(generation, link); e != nil {
		return e
	}
	defer os.Remove(link)
	if info, e := os.Lstat(s.CachePath); e == nil && info.Mode()&os.ModeSymlink == 0 {
		return problem("CACHE", "CACHE_CORRUPT", "refusing to replace an unmanaged cache path")
	}
	if e = os.Rename(link, s.CachePath); e != nil {
		return e
	}
	return nil
}
