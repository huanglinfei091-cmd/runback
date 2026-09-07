package failure

import "testing"

func TestPytestDidNotRaiseIsStructuredEvidence(t *testing.T) {
	log := `tests/test_formparser.py:61: in test_limiting
    pytest.raises(RequestEntityTooLarge, lambda: req.form["foo"])
E   Failed: DID NOT RAISE <class 'werkzeug.exceptions.RequestEntityTooLarge'>
FAILED tests/test_formparser.py::TestFormParser::test_limiting - Failed: DID ...
Process completed with exit code 1.`
	e := Parse(log, "Run tests", "pytest", nil)
	if e.Level != "TEST" || e.FailureCount != 1 || len(e.Items) != 1 {
		t.Fatalf("evidence was not structured: %+v", e)
	}
	item := e.Items[0]
	if item.Identity != "tests/test_formparser.py::TestFormParser::test_limiting" || item.Exception != "Failed" || item.File != "tests/test_formparser.py" || item.Line != 61 || item.Message != "DID NOT RAISE <class 'werkzeug.exceptions.RequestEntityTooLarge'>" {
		t.Fatalf("unexpected item: %+v", item)
	}
	m := Compare(log, log, "Run tests", "pytest", 1, true)
	if m.Status != "SAME_FAILURE" || m.MatchedCount != 1 {
		t.Fatalf("structured evidence did not match: %+v", m)
	}
}
