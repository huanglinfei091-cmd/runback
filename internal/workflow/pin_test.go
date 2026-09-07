package workflow

import "testing"

func TestUVPinUsesVersionInput(t *testing.T) {
	w, e := Parse([]byte("jobs:\n  tests:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: astral-sh/setup-uv@abc\n"))
	if e != nil {
		t.Fatal(e)
	}
	PinObserved(&w, "tests", map[string]string{"uv-version": "0.12.9"})
	job := w.Raw["jobs"].(map[string]any)["tests"].(map[string]any)
	step := job["steps"].([]any)[0].(map[string]any)
	with := step["with"].(map[string]any)
	if with["version"] != "0.12.9" || with["uv-version"] != nil {
		t.Fatal(with)
	}
}
