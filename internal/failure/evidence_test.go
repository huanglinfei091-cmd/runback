package failure

import (
	"strings"
	"testing"
)

const pytestLog = "tests/test_auth.py:81: in test_status\n    assert response.status == 200\nE   AssertionError: assert 500 == 200\nFAILED tests/test_auth.py::test_status - AssertionError: assert 500 == 200\nProcess completed with exit code 1."
const mypyLog = "2026-08-31T13:28:12Z src/app/provider.py:117: error: Unused \"type: ignore\" comment\n2026-08-31T13:28:12Z [unused-ignore]\nProcess completed with exit code 1."

func TestPytestSame(t *testing.T) {
	m := Compare(pytestLog, pytestLog, "Run tests", "pytest", 1, true)
	if m.Status != "SAME_FAILURE" || m.MatchedCount != 1 {
		t.Fatalf("%+v", m)
	}
}
func TestMypyWrappedDiagnostic(t *testing.T) {
	m := Compare(mypyLog, "src/app/provider.py:117: error: Unused \"type: ignore\" comment [unused-ignore]", "typecheck", "mypy", 1, true)
	if m.Status != "SAME_FAILURE" {
		t.Fatalf("%+v", m)
	}
}
func TestStructuredMutationCannotMatch(t *testing.T) {
	for _, pair := range [][2]string{{"assert 500 == 200", "assert 404 == 200"}, {"test_status", "test_other"}, {"AssertionError", "ValueError"}, {"test_auth.py", "test_different.py"}, {":81:", ":82:"}} {
		local := strings.ReplaceAll(pytestLog, pair[0], pair[1])
		m := Compare(pytestLog, local, "Run tests", "pytest", 1, true)
		if m.Status == "SAME_FAILURE" {
			t.Fatal(pair, m)
		}
	}
}
func TestStepOnlyIsNeverSame(t *testing.T) {
	s := "FAILED unknown case\nerror: unexpected status\nProcess completed with exit code 1."
	m := Compare(s, s, "tests", "npm test", 1, true)
	if m.Status != "LIKELY_MATCH" {
		t.Fatalf("%+v", m)
	}
}
func TestWrongStepBlocked(t *testing.T) {
	if m := Compare(pytestLog, pytestLog, "tests", "pytest", 1, false); m.Status != "REPLAY_BLOCKED" {
		t.Fatal(m)
	}
}
func TestMissingExitNotSame(t *testing.T) {
	s := strings.Split(pytestLog, "Process completed")[0]
	if m := Compare(s, s, "tests", "pytest", 1, true); m.Status == "SAME_FAILURE" {
		t.Fatal(m)
	}
}
func TestExtraFailureNotSame(t *testing.T) {
	extra := "other.py:1: error: incompatible type [arg-type]"
	if m := Compare(mypyLog, mypyLog+"\n"+extra, "tests", "mypy", 1, true); m.Status == "SAME_FAILURE" {
		t.Fatal(m)
	}
}

func TestUnparsedFailuresStillCount(t *testing.T) {
	local := pytestLog + "\nFAILED tests/test_pager.py::test_pager[less] - AssertionError\nFAILED tests/test_pager.py::test_pager[ less ] - AssertionError"
	m := Compare(pytestLog, local, "tests", "pytest", 1, true)
	if m.Status != "DIFFERENT_FAILURE" || m.RemoteCount != 1 || m.LocalCount != 3 || m.MatchedCount != 1 || len(m.Local.UnparsedFailures) != 2 {
		t.Fatalf("%+v", m)
	}
}
