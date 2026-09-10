package failure

import "testing"

func TestPyrightSameAcrossGitHubAndRunBackWorkspace(t *testing.T) {
	remote := "2026-09-09T11:40:00Z benchmarks/federation/benchmark.py:27:6 - error: Import \"scipy.stats\" could not be resolved (reportMissingImports)\nProcess completed with exit code 1."
	local := "/tmp/rb/4dcf2751bc86/workspace/benchmarks/federation/benchmark.py:27:6 - error: Import \"scipy.stats\" could not be resolved (reportMissingImports)"

	m := Compare(remote, local, "Pyright", "uv run pyright", 1, true)
	if m.Status != "SAME_FAILURE" || m.Level != "STRUCTURED" || m.RemoteCount != 1 || m.LocalCount != 1 || m.MatchedCount != 1 {
		t.Fatalf("%+v", m)
	}
	want := Item{
		Kind:      "pyright",
		Identity:  "benchmarks/federation/benchmark.py:27:pyright[reportMissingImports]",
		Exception: "pyright[reportMissingImports]",
		File:      "benchmarks/federation/benchmark.py",
		Line:      27,
		Message:   "Import \"scipy.stats\" could not be resolved",
	}
	if len(m.Remote.Items) != 1 || m.Remote.Items[0] != want || len(m.Local.Items) != 1 || m.Local.Items[0] != want {
		t.Fatalf("remote=%+v local=%+v", m.Remote.Items, m.Local.Items)
	}
}

func TestPyrightMutationCannotMatch(t *testing.T) {
	remote := "pkg/check.py:9:3 - error: Type of \"value\" is unknown (reportUnknownVariableType)\nProcess completed with exit code 1."
	local := "pkg/check.py:9:3 - error: Type of \"other\" is unknown (reportUnknownVariableType)"
	if m := Compare(remote, local, "Pyright", "pyright", 1, true); m.Status == "SAME_FAILURE" {
		t.Fatalf("mutated diagnostic matched: %+v", m)
	}
}
