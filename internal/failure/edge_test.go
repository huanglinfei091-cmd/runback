package failure

import "testing"

func TestPassingJobIsDifferent(t *testing.T) {
	m := Compare(pytestLog, "all passed", "tests", "pytest", 0, false)
	if m.Status != "DIFFERENT_FAILURE" {
		t.Fatal(m)
	}
}
func TestParameterizedEvidenceIsNotGuessed(t *testing.T) {
	s := "tests/test_auth.py:81: in test_status\nE AssertionError: bad response\nFAILED tests/test_auth.py::test_status[a] - AssertionError: bad response\nFAILED tests/test_auth.py::test_status[b] - AssertionError: bad response\nProcess completed with exit code 1."
	m := Compare(s, s, "tests", "pytest", 1, true)
	if m.Status == "SAME_FAILURE" {
		t.Fatal("one traceback cannot establish two parameterized failures")
	}
}
