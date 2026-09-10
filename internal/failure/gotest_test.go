package failure

import "testing"

func TestGoTestSameAcrossGitHubAndLocalOutput(t *testing.T) {
	remote := "2026-09-07T03:28:27.9904208Z --- FAIL: TestAudioDialogPublishesStreamingReasoningForVoiceRun (0.00s)\n2026-09-07T03:28:27.9905050Z     audio_dialog_test.go:1077: published messages = []agent.Message(nil), want one streaming reasoning message\n2026-09-07T03:28:28.0383558Z Process completed with exit code 1."
	local := "--- FAIL: TestAudioDialogPublishesStreamingReasoningForVoiceRun (0.00s)\n    audio_dialog_test.go:1077: published messages = []agent.Message(nil), want one streaming reasoning message"

	m := Compare(remote, local, "Run Go unit tests", "go test ./...", 1, true)
	if m.Status != "SAME_FAILURE" || m.Level != "TEST" || m.RemoteCount != 1 || m.LocalCount != 1 || m.MatchedCount != 1 {
		t.Fatalf("%+v", m)
	}
	want := Item{
		Kind:      "go-test",
		Identity:  "TestAudioDialogPublishesStreamingReasoningForVoiceRun",
		Exception: "go-test",
		File:      "audio_dialog_test.go",
		Line:      1077,
		Message:   "published messages = []agent.Message(nil), want one streaming reasoning message",
	}
	if len(m.Remote.Items) != 1 || m.Remote.Items[0] != want || len(m.Local.Items) != 1 || m.Local.Items[0] != want {
		t.Fatalf("remote=%+v local=%+v", m.Remote.Items, m.Local.Items)
	}
}

func TestGoTestMessageMutationCannotMatch(t *testing.T) {
	remote := "--- FAIL: TestThing (0.00s)\n    thing_test.go:42: got 1, want 2\nProcess completed with exit code 1."
	local := "--- FAIL: TestThing (0.00s)\n    thing_test.go:42: got 3, want 2"
	if m := Compare(remote, local, "test", "go test", 1, true); m.Status == "SAME_FAILURE" {
		t.Fatalf("mutated diagnostic matched: %+v", m)
	}
}

func TestGoTestWithoutSourceDiagnosticRemainsInsufficient(t *testing.T) {
	log := "--- FAIL: TestThing (0.00s)\nProcess completed with exit code 1."
	m := Compare(log, log, "test", "go test", 1, true)
	if m.Status == "SAME_FAILURE" || m.RemoteCount != 1 || len(m.Remote.UnparsedFailures) != 1 {
		t.Fatalf("%+v", m)
	}
}
