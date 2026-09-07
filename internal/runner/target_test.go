package runner

import (
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"testing"
)

func TestSkippedTargetIsNotVerifiedSuccess(t *testing.T) {
	l := lockfile.Lock{JobID: "tests", Failure: lockfile.Failure{Name: "Run pytest"}}
	for _, log := range []string{
		`{"jobID":"tests","step":"pytest","msg":"Skipping step pytest"}`,
		`{"jobID":"other","step":"pytest","msg":"Success - Main pytest"}`,
		`{"jobID":"tests","stage":"Post","step":"pytest","msg":"Success - Main pytest"}`,
	} {
		if ActStepSucceeded(log, l) {
			t.Fatal(log)
		}
	}
	if !ActStepSucceeded(`{"jobID":"tests","stage":"Main","step":"pytest","msg":"Success - Main pytest"}`, l) {
		t.Fatal("target success missed")
	}
}
