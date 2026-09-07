package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Bundle is exported by Windows and copied to an unauthenticated Ubuntu lab.
type RunEvidence struct {
	URL     string            `json:"url"`
	RunData Run               `json:"run"`
	JobData []Job             `json:"jobs"`
	Files   map[string]string `json:"files"`
	Logs    map[string]string `json:"logs"`
}

// Bundle and online acquisition share exactly this canonical evidence representation.
type Bundle = RunEvidence

func ReadBundle(path string) (*Bundle, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var f Bundle
	e = json.Unmarshal(b, &f)
	return &f, e
}
func (f *Bundle) Run(_ context.Context, t Target) (Run, error) {
	if t.RunID != f.RunData.ID || !equalRepo(t.Repository(), f.RunData.Repository.FullName) {
		return Run{}, fmt.Errorf("bundle URL mismatch")
	}
	if t.Attempt > 0 && t.Attempt != f.RunData.RunAttempt {
		return Run{}, fmt.Errorf("bundle attempt mismatch")
	}
	return f.RunData, nil
}
func equalRepo(a, b string) bool { return a == b }
func (f *Bundle) Jobs(_ context.Context, _ Target, a int) ([]Job, error) {
	if a != f.RunData.RunAttempt {
		return nil, fmt.Errorf("bundle attempt mismatch")
	}
	return f.JobData, nil
}
func (f *Bundle) File(_ context.Context, repo, path, sha string) ([]byte, error) {
	s, ok := f.Files[repo+"@"+sha+":"+path]
	if !ok {
		return nil, fmt.Errorf("workflow missing from evidence bundle at %s", sha)
	}
	return []byte(s), nil
}
func (f *Bundle) Get(_ context.Context, path string) ([]byte, error) {
	s, ok := f.Logs[path]
	if !ok {
		return nil, fmt.Errorf("logs missing from evidence bundle")
	}
	return []byte(s), nil
}
func (f *Bundle) Save(path string) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(f, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0600)
}
