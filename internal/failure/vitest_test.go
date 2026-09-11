package failure

import (
	"strings"
	"testing"
)

const vitestLayoutFailure = `FAIL  tests/vitest/layout.test.ts > footer edit link for route / points at /
AssertionError: expected edit link to point at readme // Object.is equality
Expected: "https://example.test/blob/-/readme.md"
Received: "https://example.test/blob/-/src/routes/+page.svelte"
 ❯ tests/vitest/layout.test.ts:44:41`

const vitestTOCHeader1 = `FAIL  tests/vitest/toc.svelte.test.ts > Toc > static metadata keeps matching item indexes for id=sec:1`
const vitestTOCHeader2 = `FAIL  tests/vitest/toc.svelte.test.ts > Toc > static metadata keeps matching item indexes for id=123`
const vitestTOCHeader3 = `FAIL  tests/vitest/toc.svelte.test.ts > Toc > static metadata keeps matching item indexes for id=part.one`
const vitestTOCBody = `AssertionError: No element found for selector: aside.toc li.active
 ❯ doc_query tests/vitest/index.ts:18:3
 ❯ tests/vitest/toc.svelte.test.ts:352:14`

func vitestFourFailures() string {
	grouped := strings.Join([]string{vitestTOCHeader1, vitestTOCHeader2, vitestTOCHeader3, vitestTOCBody}, "\n")
	return strings.Join([]string{vitestLayoutFailure, grouped, "Process completed with exit code 1."}, "\n⎯⎯⎯\n")
}

func TestVitestFourFailuresAreStrictTestEvidence(t *testing.T) {
	remote := vitestFourFailures()
	local := strings.ReplaceAll(remote, "❯ tests/", "❯ /tmp/rb/abc123/workspace/tests/")
	m := Compare(remote, local, "Unit tests with coverage", "npm run test:coverage", 1, true)
	if m.Status != "SAME_FAILURE" || m.Level != "TEST" || m.RemoteCount != 4 || m.LocalCount != 4 || m.MatchedCount != 4 {
		t.Fatalf("unexpected match: %+v", m)
	}
	want := Item{
		Kind:      "vitest",
		Identity:  "vitest:tests/vitest/layout.test.ts:44:41:footer edit link for route / points at /",
		Exception: "AssertionError",
		File:      "tests/vitest/layout.test.ts",
		Line:      44,
		Message:   "expected edit link to point at readme // Object.is equality\nExpected: \"https://example.test/blob/-/readme.md\"\nReceived: \"https://example.test/blob/-/src/routes/+page.svelte\"",
	}
	if m.Remote.Items[0] != want {
		t.Fatalf("unexpected first item: %#v", m.Remote.Items[0])
	}
}

func TestVitestStructuredMutationsCannotMatch(t *testing.T) {
	remote := vitestFourFailures()
	mutations := [][2]string{
		{"points at /", "points at /docs"},
		{"layout.test.ts", "other.test.ts"},
		{"44:41", "45:41"},
		{"44:41", "44:42"},
		{"AssertionError", "TypeError"},
		{"expected edit link", "expected footer link"},
		{"/-/readme.md", "/-/README.md"},
		{"/src/routes/+page.svelte", "/src/routes/+layout.svelte"},
	}
	for _, mutation := range mutations {
		local := strings.Replace(remote, mutation[0], mutation[1], 1)
		if got := Compare(remote, local, "unit", "vitest", 1, true); got.Status == "SAME_FAILURE" {
			t.Fatalf("accepted mutation %q -> %q: %+v", mutation[0], mutation[1], got)
		}
	}
	if got := Compare(remote, remote, "unit", "vitest", 2, true); got.Status == "SAME_FAILURE" {
		t.Fatal("accepted changed exit code", got)
	}
}

func TestVitestMissingExtraAndIncompleteFailuresCannotMatch(t *testing.T) {
	remote := vitestFourFailures()
	missing := strings.Replace(remote, vitestTOCHeader3+"\n", "", 1)
	extra := remote + "\n⎯⎯⎯\n" + strings.ReplaceAll(vitestTOCHeader3, "part.one", "extra") + "\n" + vitestTOCBody
	noLocation := strings.Replace(remote, " ❯ tests/vitest/layout.test.ts:44:41", " ❯ tests/vitest/helper.ts:44:41", 1)
	for name, local := range map[string]string{"missing": missing, "extra": extra, "no matching location": noLocation} {
		if got := Compare(remote, local, "unit", "vitest", 1, true); got.Status == "SAME_FAILURE" {
			t.Fatalf("accepted %s evidence: %+v", name, got)
		}
	}
	parsed := Parse(noLocation, "unit", "vitest", nil)
	if parsed.Level != "STEP" || parsed.FailureCount != 4 || len(parsed.UnparsedFailures) != 1 {
		t.Fatalf("incomplete block was not retained as unparsed: %+v", parsed)
	}
}

func TestVitestDuplicateIdentityIsUnparsed(t *testing.T) {
	log := vitestLayoutFailure + "\n⎯⎯⎯\n" + vitestLayoutFailure + "\nProcess completed with exit code 1."
	m := Compare(log, log, "unit", "vitest", 1, true)
	if m.Status == "SAME_FAILURE" || m.RemoteCount != 2 || len(m.Remote.Items) != 0 || len(m.Remote.UnparsedFailures) != 2 {
		t.Fatalf("duplicate identity was accepted: %+v", m)
	}
}

func TestVitestANSIAndTimestampNormalizeAcrossWorkspaces(t *testing.T) {
	remote := "2026-09-10T07:00:00Z \x1b[41m\x1b[1m FAIL \x1b[22m\x1b[49m tests/sample.test.ts\x1b[2m > \x1b[22mworks\n" +
		"2026-09-10T07:00:01Z \x1b[31m\x1b[1mAssertionError\x1b[22m: expected 1 to be 2\x1b[39m\n" +
		"2026-09-10T07:00:02Z  \x1b[2m❯\x1b[22m tests/sample.test.ts:\x1b[2m9:7\x1b[22m\x1b[39m\n" +
		"2026-09-10T07:00:03Z Process completed with exit code 1."
	local := "FAIL tests/sample.test.ts > works\nAssertionError: expected 1 to be 2\n❯ /tmp/rb/a1b2/workspace/tests/sample.test.ts:9:7"
	if got := Compare(remote, local, "unit", "vitest", 1, true); got.Status != "SAME_FAILURE" || got.MatchedCount != 1 {
		t.Fatalf("normalized Vitest evidence did not match: %+v", got)
	}
}

func TestVitestUnsupportedHeaderCountsButCannotMatch(t *testing.T) {
	log := "FAIL tests/component.test.vue > renders\nAssertionError: expected true\n❯ tests/component.test.vue:4:2\nProcess completed with exit code 1."
	e := Parse(log, "unit", "vitest", nil)
	if e.FailureCount != 1 || e.Level != "STEP" || len(e.Items) != 0 || len(e.UnparsedFailures) != 1 {
		t.Fatalf("unsupported Vitest block was not retained: %+v", e)
	}
	if got := Compare(log, log, "unit", "vitest", 1, true); got.Status == "SAME_FAILURE" {
		t.Fatal("unsupported block matched", got)
	}
}
