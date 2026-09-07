package lockfile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"os"
	"path/filepath"
	"regexp"
)

type Failure struct {
	Shell            string         `json:"shell"`
	WorkingDirectory string         `json:"working_directory"`
	Environment      map[string]any `json:"environment"`
	Name             string         `json:"name"`
	Command          string         `json:"command"`
	StepIndex        int            `json:"step_index"`
	Log              string         `json:"log"`
}
type Lock struct {
	Network          string            `json:"network,omitempty"`
	NetworkSource    string            `json:"network_source,omitempty"`
	ObservedVersions map[string]string `json:"observed_versions"`
	Version          int               `json:"version"`
	URL              string            `json:"url"`
	Repository       string            `json:"repository"`
	RunID            int64             `json:"run_id"`
	Attempt          int               `json:"attempt"`
	HeadSHA          string            `json:"head_sha"`
	Commit           string            `json:"commit"`
	CheckoutSource   string            `json:"checkout_source"`
	Branch           string            `json:"branch"`
	Event            string            `json:"event"`
	Actor            string            `json:"actor"`
	WorkflowPath     string            `json:"workflow_path"`
	WorkflowRef      string            `json:"workflow_ref"`
	Workflow         string            `json:"workflow"`
	JobID            string            `json:"job_id"`
	JobName          string            `json:"job_name"`
	JobNumber        int64             `json:"job_number"`
	Matrix           map[string]any    `json:"matrix"`
	Runner           string            `json:"runner"`
	Image            string            `json:"image"`
	ToolVersions     map[string]string `json:"tool_versions"`
	Failure          Failure           `json:"failure"`
	EventJSON        json.RawMessage   `json:"event_json"`
	Warnings         []string          `json:"warnings"`
	Blockers         []string          `json:"blockers"`
}

var SHA = regexp.MustCompile("^[a-f0-9]{40}$")
var jobID = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_-]*$")

func (l Lock) Key() string { return fmt.Sprintf("runback-%d-%d-%d", l.RunID, l.Attempt, l.JobNumber) }
func (l Lock) Validate() error {
	t, e := github.ParseURL(l.URL)
	if e != nil {
		return e
	}
	if l.Version != 1 || t.Repository() != l.Repository || t.RunID != l.RunID || l.Attempt < 1 || l.JobNumber < 1 {
		return fmt.Errorf("invalid lock identity/version")
	}
	if !SHA.MatchString(l.Commit) || !SHA.MatchString(l.HeadSHA) {
		return fmt.Errorf("lock requires full commit SHAs")
	}
	if !jobID.MatchString(l.JobID) {
		return fmt.Errorf("invalid job ID")
	}
	if l.Event != "push" && l.Event != "pull_request" {
		return fmt.Errorf("unsupported event")
	}
	if !json.Valid(l.EventJSON) {
		return fmt.Errorf("invalid event JSON")
	}
	return nil
}
func Read(path string) (Lock, error) {
	var l Lock
	b, e := os.ReadFile(path)
	if e != nil {
		return l, e
	}
	if e = json.Unmarshal(b, &l); e != nil {
		return l, e
	}
	return l, l.Validate()
}
func Write(path string, l Lock) error {
	b, e := json.MarshalIndent(l, "", "  ")
	if e != nil {
		return e
	}
	return Atomic(path, append(b, '\n'))
}
func Atomic(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".runback-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func Digest(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }
