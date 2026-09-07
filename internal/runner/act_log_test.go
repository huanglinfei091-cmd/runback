package runner

import (
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"testing"
)

func TestActualActNaming(t *testing.T) {
	l := lockfile.Lock{JobID: "typing", Failure: lockfile.Failure{Name: "Run uv run tox -e typing", Command: "uv run tox -e typing"}}
	log := `{"jobID":"typing","step":"uv run tox -e typing","stage":"Main","msg":"src/app.py:1: error: bad type [arg-type]"}
{"jobID":"typing","step":"uv run tox -e typing","stage":"Main","stepResult":"failure","msg":"  ❌  Failure - Main uv run tox -e typing [1s]"}
`
	s, failed := ActStepLog(log, l)
	if !failed || s == "" {
		t.Fatal(s, failed)
	}
}
func TestPreStageNeverCounts(t *testing.T) {
	l := lockfile.Lock{JobID: "typing", Failure: lockfile.Failure{Name: "Run mypy"}}
	log := `{"jobID":"typing","step":"mypy","stage":"Pre","msg":"Failure - Main mypy"}
`
	s, failed := ActStepLog(log, l)
	if failed || s != "" {
		t.Fatal(s, failed)
	}
}
