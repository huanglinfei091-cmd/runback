package runner

import (
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"strings"
	"testing"
)

func TestActStepIsolation(t *testing.T) {
	l := lockfile.Lock{JobID: "tests", Failure: lockfile.Failure{Name: "Run tests", StepIndex: 2}}
	log := `{"jobID":"other","step":"Run tests","msg":"error: wrong job"}
{"jobID":"tests","step":"0","msg":"error: setup failed"}
{"jobID":"tests","step":"Run tests","msg":"AssertionError: expected 1"}
{"jobID":"tests","step":"Run tests","msg":"FAILED tests/a.py"}
{"jobID":"tests","step":"Run tests","msg":"  ❌  Failure - Main Run tests"}
`
	out, failed := ActStepLog(log, l)
	if !failed || strings.Contains(out, "wrong job") || strings.Contains(out, "setup failed") {
		t.Fatal(out, failed)
	}
}
func TestCredentialEnvNotForwarded(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "should-never-leak")
	t.Setenv("GH_TOKEN", "also-secret")
	t.Setenv("RUNBACK_GITHUB_TOKEN", "acquisition-only-secret")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	for _, v := range env(t.TempDir()) {
		if strings.Contains(v, "should-never-leak") || strings.Contains(v, "also-secret") || strings.Contains(v, "AWS_SECRET_ACCESS_KEY") || strings.Contains(v, "acquisition-only-secret") {
			t.Fatal("credential forwarded")
		}
	}
}

func TestActionFetchFailureClassification(t *testing.T) {
	log := `failed to fetch "https://github.com/astral-sh/setup-uv" version "abc": connection reset by peer`
	if got := ExecutorCause(log); got != "ACTION_FETCH_FAILED" {
		t.Fatalf("got %s", got)
	}
	if got := ExecutorCause("dial: connection reset by peer"); got != "NETWORK_DEPENDENCY" {
		t.Fatalf("got %s", got)
	}
}
