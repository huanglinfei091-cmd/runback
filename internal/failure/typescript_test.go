package failure

import (
	"strings"
	"testing"
)

func TestTypeScriptDiagnostics(t *testing.T) {
	remote := "2026-09-10T02:00:00Z src/view.tsx(10,22): error TS2551: Property 'title' does not exist.\n" +
		"src/view.tsx(10,38): error TS7031: Binding element 'key' implicitly has an 'any' type.\n" +
		"src/view.tsx(10,50): error TS7031: Binding element 'value' implicitly has an 'any' type.\nProcess completed with exit code 1."
	local := strings.ReplaceAll(remote, "src/view.tsx", "/tmp/rb/abc123/workspace/src/view.tsx")
	m := Compare(remote, local, "Build", "yarn build", 1, true)
	if m.Status != "SAME_FAILURE" || m.Level != "STRUCTURED" || m.RemoteCount != 3 || m.LocalCount != 3 || m.MatchedCount != 3 {
		t.Fatalf("unexpected match: %+v", m)
	}
	for _, change := range [][2]string{
		{"(10,22)", "(10,23)"}, {"(10,22)", "(11,22)"}, {"TS2551", "TS2552"},
		{"Property 'title'", "Property 'name'"}, {"src/view.tsx", "src/other.tsx"},
	} {
		if got := Compare(remote, strings.ReplaceAll(local, change[0], change[1]), "Build", "yarn build", 1, true); got.Status == "SAME_FAILURE" {
			t.Fatalf("accepted changed diagnostic %v", change)
		}
	}
	if got := Compare(remote, local+"\nsrc/new.ts(1,1): error TS2304: Cannot find name 'x'.", "Build", "yarn build", 1, true); got.Status != "DIFFERENT_FAILURE" {
		t.Fatal("extra local error accepted", got)
	}
	if Compare(remote, local, "Build", "yarn build", 2, true).Status == "SAME_FAILURE" || Compare(remote, local, "Build", "yarn build", 1, false).Status == "SAME_FAILURE" {
		t.Fatal("exit or target-step mismatch accepted")
	}
}

func TestTypeScriptIncompleteDiagnosticIsNotStructured(t *testing.T) {
	for _, log := range []string{
		"error TS2304: Cannot find name 'x'.", "src/x.ts(1,1): error: missing code", "Error: found typescript issues",
	} {
		if got := Parse(log, "Build", "tsc", nil); got.Level != "STEP" || len(got.Items) != 0 {
			t.Fatal("incomplete diagnostic accepted", got)
		}
	}
	log := "src/x.ts(1,1): error TS2304: Cannot find name 'x'.\nerror TS18003: No inputs were found.\nProcess completed with exit code 1."
	if got := Compare(log, log, "Build", "tsc", 1, true); got.Status == "SAME_FAILURE" || got.RemoteCount != 2 {
		t.Fatal("partial diagnostic set accepted", got)
	}
}

func TestTypeScriptContinuationMustAgree(t *testing.T) {
	remote := "2026-09-10T02:00:00Z src/x.ts(1,2): error TS2322: Type 'A' is not assignable to type 'B'.\n2026-09-10T02:00:00Z   Property 'name' is missing.\nProcess completed with exit code 1."
	local := "src/x.ts(1,2): error TS2322: Type 'A' is not assignable to type 'B'.\n  Property 'name' is missing.\n"
	if got := Compare(remote, local, "Build", "tsc", 1, true); got.Status != "SAME_FAILURE" {
		t.Fatal(got)
	}
	if got := Compare(remote, strings.ReplaceAll(local, "'name'", "'id'"), "Build", "tsc", 1, true); got.Status == "SAME_FAILURE" {
		t.Fatal("different continuation accepted", got)
	}
}
