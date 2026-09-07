package failure

import (
	"os"
	"strings"
	"testing"
)

func TestRealFlaskDiagnostic(t *testing.T) {
	b, e := os.ReadFile("../../testdata/real/flask-failure.log")
	if e != nil {
		t.Fatal(e)
	}
	r := Parse(string(b), "typing", "mypy", nil)
	if r.Level != "STRUCTURED" || len(r.Items) != 1 {
		t.Fatalf("%+v", r)
	}
	v := r.Items[0]
	if v.File != "src/flask/json/provider.py" || v.Line != 117 || v.Exception != "mypy[unused-ignore]" || v.Message != "Unused \"type: ignore\" comment" {
		t.Fatalf("%+v", v)
	}
	local := "src/flask/json/provider.py:117: error: Unused \"type: ignore\" comment [unused-ignore]"
	if m := Compare(string(b), local, "typing", "mypy", 1, true); m.Status != "SAME_FAILURE" {
		t.Fatalf("%+v", m)
	}
	local = strings.ReplaceAll(local, "117", "118")
	if m := Compare(string(b), local, "typing", "mypy", 1, true); m.Status == "SAME_FAILURE" {
		t.Fatal("location mismatch accepted")
	}
}
