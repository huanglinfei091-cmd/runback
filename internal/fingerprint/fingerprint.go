package fingerprint

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
var timestamp = regexp.MustCompile(`^\d{4}-\d\d-\d\dT[0-9:.]+Z\s*`)
var location = regexp.MustCompile(`(?:/home/runner/work/[^/]+/[^/]+/|/github/workspace/|/runback/workspace/|/tmp/rb/[A-Za-z0-9._-]+/workspace/)`)
var evidence = regexp.MustCompile(`(?i)(\bFAIL(?:ED)?\b|\b(?:Type|Value|Assertion|Runtime|Import|ModuleNotFound|Syntax|Reference)Error\b|\berror:|\bpanic:|Expected:|Received:|AssertionError|assert .+|--- FAIL:|E\s+\w+Error|\bNo solution found\b|\bbecause\b.*\b(?:requires|depends)\b)`)
var passedTest = regexp.MustCompile(`\s(?:PASSED|SKIPPED|XFAIL)\s*(?:\[.*\])?$`)
var generic = regexp.MustCompile(`(?i)(process completed with exit code|exit status \d+|job failed|step failed|failure - main|error: exit with|some checks were not successful|evaluation failed|FAIL code [0-9]|[0-9]+ failed,|[0-9]+ passed,)`)

type Print struct {
	Hash  string   `json:"hash"`
	Lines []string `json:"lines"`
}

func Normalize(s string) string {
	s = ansi.ReplaceAllString(s, "")
	s = timestamp.ReplaceAllString(s, "")
	if strings.HasPrefix(s, "[") {
		if i := strings.Index(s, "]"); i >= 0 {
			rest := strings.TrimSpace(s[i+1:])
			if strings.HasPrefix(rest, "|") {
				s = strings.TrimPrefix(rest, "|")
			}
		}
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "##[error]"))
	s = location.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
func Build(log string) Print {
	set := map[string]bool{}
	for _, l := range strings.Split(log, "\n") {
		l = Normalize(l)
		if len(l) > 1000 {
			continue
		}
		if evidence.MatchString(l) && !generic.MatchString(l) && !passedTest.MatchString(l) && !strings.HasPrefix(l, "echo ") {
			set[l] = true
		}
	}
	lines := []string{}
	for l := range set {
		lines = append(lines, l)
	}
	sort.Strings(lines)
	hash := ""
	if len(lines) > 0 {
		hash = fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(lines, "\n"))))
	}
	return Print{hash, lines}
}

type Comparison struct {
	Status string  `json:"status"`
	Score  float64 `json:"similarity"`
	Remote Print   `json:"remote"`
	Local  Print   `json:"local"`
	Reason string  `json:"reason"`
}

func Compare(remote, local string, exit int, stepFailed bool) Comparison {
	r, l := Build(remote), Build(local)
	c := Comparison{Status: "unverified", Remote: r, Local: l}
	if exit == 0 {
		c.Status = "not_reproduced"
		c.Reason = "local execution succeeded"
		return c
	}
	if len(r.Lines) < 2 || len(l.Lines) < 2 {
		c.Reason = "need at least two distinctive error lines on each side"
		return c
	}
	set := map[string]bool{}
	for _, s := range r.Lines {
		set[s] = true
	}
	n := 0
	for _, s := range l.Lines {
		if set[s] {
			n++
		}
	}
	c.Score = float64(n) / float64(len(r.Lines)+len(l.Lines)-n)
	if !stepFailed {
		c.Reason = "act did not report failure in the selected step"
		return c
	}
	if c.Score >= 0.8 {
		c.Status = "likely_match"
		c.Reason = "selected step failed with matching error signature (heuristic, not proof of identical environment)"
	} else {
		c.Status = "different_failure"
		c.Reason = "local error signature differs"
	}
	return c
}

// StepLog isolates the remote failed step using timestamps supplied by the jobs API.
func StepLog(log, start, end string) string {
	a, e1 := time.Parse(time.RFC3339Nano, start)
	b, e2 := time.Parse(time.RFC3339Nano, end)
	if e1 != nil || e2 != nil {
		return ""
	}
	var lines []string
	for _, l := range strings.Split(log, "\n") {
		i := strings.IndexByte(l, ' ')
		if i < 0 {
			continue
		}
		t, e := time.Parse(time.RFC3339Nano, l[:i])
		if e == nil && !t.Before(a) && t.Before(b.Add(time.Second)) {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}
