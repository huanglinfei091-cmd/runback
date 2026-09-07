package runner

import (
	"encoding/json"
	"github.com/huanglinfei091-cmd/runback/internal/failure"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"os"
	"strconv"
	"testing"
)

// This test consumes a captured real FULL_JOB execution, never a generated toy failure.
func TestCapturedFullJobEvidence(t *testing.T) {
	path := os.Getenv("RUNBACK_CASE_LOCK")
	if path == "" {
		t.Skip("set RUNBACK_CASE_LOCK, RUNBACK_LOCAL_LOG and RUNBACK_ACT_EXIT to validate a real act capture")
	}
	l, e := lockfile.Read(path)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(os.Getenv("RUNBACK_LOCAL_LOG"))
	if e != nil {
		t.Fatal(e)
	}
	exit, e := strconv.Atoi(os.Getenv("RUNBACK_ACT_EXIT"))
	if e != nil {
		t.Fatal(e)
	}
	local, failed := ActStepLog(string(b), l)
	m := failure.Compare(l.Failure.Log, local, l.Failure.Name, l.Failure.Command, exit, failed)
	data, _ := json.MarshalIndent(m, "", "  ")
	t.Log(string(data))
	if out := os.Getenv("RUNBACK_EVIDENCE_OUTPUT"); out != "" {
		if e = os.WriteFile(out, data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	expected := os.Getenv("RUNBACK_EXPECT")
	if expected == "" {
		expected = "SAME_FAILURE"
	}
	if m.Status != expected {
		t.Fatalf("real execution did not match: %s", m.Status)
	}
}
