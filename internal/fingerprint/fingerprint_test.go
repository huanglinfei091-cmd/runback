package fingerprint

import (
	"testing"
)

const remote = "2026-09-03T13:21:19Z FAILED tests/test_auth.py::test_status\n2026-09-03T13:21:19Z AssertionError: Expected: 200 Received: 500"

func TestMatch(t *testing.T) {
	c := Compare(remote, "FAILED tests/test_auth.py::test_status\nAssertionError: Expected: 200 Received: 500", 1, true)
	if c.Status != "likely_match" || c.Score != 1 {
		t.Fatalf("%+v", c)
	}
}
func TestDifferentNonzero(t *testing.T) {
	c := Compare(remote, "error: image pull failed\nerror: network timeout", 1, true)
	if c.Status == "likely_match" {
		t.Fatal(c)
	}
}
func TestGenericExitCannotMatch(t *testing.T) {
	c := Compare("Process completed with exit code 1.", "Process completed with exit code 1.", 1, true)
	if c.Status != "unverified" {
		t.Fatal(c)
	}
}
func TestWrongStepCannotMatch(t *testing.T) {
	c := Compare(remote, remote, 1, false)
	if c.Status == "likely_match" {
		t.Fatal(c)
	}
}
func TestPassingLocalCannotMatch(t *testing.T) {
	c := Compare(remote, remote, 0, true)
	if c.Status != "not_reproduced" {
		t.Fatal(c)
	}
}
func TestTimestampIsolation(t *testing.T) {
	s := StepLog("2026-09-03T13:21:18Z setup\n2026-09-03T13:21:19.5Z error\n2026-09-03T13:21:21Z cleanup", "2026-09-03T13:21:19Z", "2026-09-03T13:21:20Z")
	if s != "2026-09-03T13:21:19.5Z error" {
		t.Fatal(s)
	}
}

func TestRunBackWorkspacePathIsNormalized(t *testing.T) {
	remote := `benchmarks/federation/benchmark.py:27:6 - error: Import "scipy.stats" could not be resolved (reportMissingImports)`
	local := `/tmp/rb/4dcf2751bc86/workspace/benchmarks/federation/benchmark.py:27:6 - error: Import "scipy.stats" could not be resolved (reportMissingImports)`
	if got := Normalize(local); got != remote {
		t.Fatalf("got %q, want %q", got, remote)
	}
}

func TestPassingParameterizedErrorsAreNotFailureEvidence(t *testing.T) {
	p := Build("tests/test_args.py::test_args[Error: bad argument] PASSED [ 10%]\nE AssertionError: 1 != 2")
	if len(p.Lines) != 1 || p.Lines[0] != "E AssertionError: 1 != 2" {
		t.Fatal(p)
	}
}
