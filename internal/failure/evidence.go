// Package failure distinguishes step similarity from structured failure identity.
package failure

import (
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/fingerprint"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Item struct {
	Kind      string `json:"kind"`
	Identity  string `json:"identity"`
	Exception string `json:"exception"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Message   string `json:"message"`
}
type Evidence struct {
	FailureCount     int      `json:"failure_count"`
	UnparsedFailures []string `json:"unparsed_failures,omitempty"`
	Level            string   `json:"level"`
	Step             string   `json:"step"`
	Command          string   `json:"command"`
	Exit             *int     `json:"exit"`
	Excerpt          []string `json:"normalized_excerpt"`
	Items            []Item   `json:"items"`
}
type Fields struct{ Identity, Exception, File, Line, Message, Exit bool }
type Match struct {
	Status        string   `json:"status"`
	Level         string   `json:"evidence_level"`
	Reason        string   `json:"reason"`
	Remote        Evidence `json:"remote"`
	Local         Evidence `json:"local"`
	MatchedFields Fields   `json:"matched_fields"`
	RemoteCount   int      `json:"remote_failures"`
	LocalCount    int      `json:"local_failures"`
	MatchedCount  int      `json:"matched_failures"`
}

var mypy = regexp.MustCompile(`^(.+?\.pyi?):(\d+):(?:\d+:)?\s*error:\s*(.+?)(?:\s+\[([a-zA-Z0-9_-]+)\])?$`)
var pyright = regexp.MustCompile(`^(.+?\.pyi?):(\d+):(\d+)\s+-\s+error:\s+(.+?)(?:\s+\(([A-Za-z][A-Za-z0-9_-]+)\))?$`)
var diagnosticCode = regexp.MustCompile(`^\[([a-zA-Z0-9_-]+)\]$`)
var frame = regexp.MustCompile(`^(.+?\.py):(\d+): in ([A-Za-z0-9_]+)$`)
var exception = regexp.MustCompile(`^E\s+((?:[A-Za-z0-9_.]*(?:Error|Exception))|Failed)(?::\s*(.*))?$`)
var goTestFailure = regexp.MustCompile(`^--- FAIL: ([^\s(]+) \([^)]+\)$`)
var goTestDiagnostic = regexp.MustCompile(`^(.+?_test\.go):(\d+):\s+(.+)$`)
var remoteExit = regexp.MustCompile(`Process completed with exit code (\d+)`)
var uuid = regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
var temp = regexp.MustCompile(`(?:/tmp|/home/runner/work/_temp)/[^ /:]+`)

func Normalize(s string) string {
	s = fingerprint.Normalize(s)
	s = uuid.ReplaceAllString(s, "<uuid>")
	s = temp.ReplaceAllString(s, "<temp>")
	return strings.TrimSpace(s)
}
func Parse(log, step, command string, exit *int) Evidence {
	e := Evidence{Level: "STEP", Step: step, Command: command, Exit: exit, Items: []Item{}, Excerpt: fingerprint.Build(log).Lines}
	if exit == nil {
		if m := remoteExit.FindStringSubmatch(log); m != nil {
			n, _ := strconv.Atoi(m[1])
			e.Exit = &n
		}
	}
	lines := strings.Split(log, "\n")
	var diagnostics []Item
	type traceback struct {
		Item
		Function string
	}
	var traces []traceback
	active := -1
	var nodes []string
	type goFailure struct {
		Item
		Name string
	}
	var goFailures []goFailure
	activeGo := -1
	for _, line := range lines {
		line = Normalize(line)
		if m := pyright.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[2])
			diagnostic := "pyright"
			if m[5] != "" {
				diagnostic += "[" + m[5] + "]"
			}
			diagnostics = append(diagnostics, Item{Kind: "pyright", File: m[1], Line: n, Message: strings.TrimSpace(m[4]), Exception: diagnostic})
			continue
		}
		if m := mypy.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[2])
			diagnostics = append(diagnostics, Item{Kind: "mypy", File: m[1], Line: n, Message: strings.TrimSpace(m[3]), Exception: m[4]})
			continue
		}
		if m := diagnosticCode.FindStringSubmatch(line); m != nil && len(diagnostics) > 0 {
			last := &diagnostics[len(diagnostics)-1]
			if last.Exception == "" {
				last.Exception = m[1]
			}
			continue
		}
		if m := frame.FindStringSubmatch(line); m != nil && strings.HasPrefix(m[3], "test_") {
			n, _ := strconv.Atoi(m[2])
			traces = append(traces, traceback{Item: Item{Kind: "pytest", File: m[1], Line: n}, Function: m[3]})
			active = len(traces) - 1
			continue
		}
		if m := exception.FindStringSubmatch(line); m != nil && active >= 0 {
			traces[active].Exception = m[1]
			traces[active].Message = Normalize(m[2])
			continue
		}
		if m := goTestFailure.FindStringSubmatch(line); m != nil {
			goFailures = append(goFailures, goFailure{Item: Item{Kind: "go-test", Identity: m[1], Exception: "go-test"}, Name: m[1]})
			activeGo = len(goFailures) - 1
			continue
		}
		if m := goTestDiagnostic.FindStringSubmatch(line); m != nil && activeGo >= 0 && goFailures[activeGo].File == "" {
			n, _ := strconv.Atoi(m[2])
			goFailures[activeGo].File = m[1]
			goFailures[activeGo].Line = n
			goFailures[activeGo].Message = strings.TrimSpace(m[3])
			continue
		}
		if strings.HasPrefix(line, "FAILED ") {
			node := strings.SplitN(strings.TrimPrefix(line, "FAILED "), " - ", 2)[0]
			if strings.Contains(node, ".py::") {
				nodes = append(nodes, node)
			}
		}
	}
	e.FailureCount = len(diagnostics) + len(nodes) + len(goFailures)
	for _, d := range diagnostics {
		if d.Exception != "" {
			if d.Kind == "mypy" {
				d.Exception = "mypy[" + d.Exception + "]"
			}
			d.Identity = fmt.Sprintf("%s:%d:%s", d.File, d.Line, d.Exception)
			e.Items = append(e.Items, d)
		}
	}
	nodeFunctions := map[string]int{}
	for _, node := range nodes {
		parts := strings.Split(node, "::")
		key := parts[0] + "::" + strings.SplitN(parts[len(parts)-1], "[", 2)[0]
		nodeFunctions[key]++
	}
	for _, node := range nodes {
		parts := strings.Split(node, "::")
		file := parts[0]
		function := strings.SplitN(parts[len(parts)-1], "[", 2)[0]
		if nodeFunctions[file+"::"+function] != 1 {
			e.UnparsedFailures = append(e.UnparsedFailures, node)
			continue
		}
		var candidates []Item
		for _, tr := range traces {
			if tr.File == file && tr.Function == function && tr.Exception != "" && tr.Message != "" {
				v := tr.Item
				v.Identity = node
				candidates = append(candidates, v)
			}
		}
		// Do not guess which traceback belongs to multiple parameterized failures.
		if len(candidates) == 1 {
			e.Items = append(e.Items, candidates[0])
		} else {
			e.UnparsedFailures = append(e.UnparsedFailures, node)
		}
	}
	for _, g := range goFailures {
		if g.File == "" || g.Line == 0 || g.Message == "" {
			e.UnparsedFailures = append(e.UnparsedFailures, "go test: "+g.Name)
			continue
		}
		e.Items = append(e.Items, g.Item)
	}
	if len(e.Items) > 0 && len(e.Items) == len(diagnostics)+len(nodes)+len(goFailures) {
		e.Level = "TEST"
		if len(nodes) == 0 && len(goFailures) == 0 {
			e.Level = "STRUCTURED"
		}
	}
	sort.Slice(e.Items, func(i, j int) bool { return e.Items[i].Identity < e.Items[j].Identity })
	return e
}
func same(a, b Item) Fields {
	return Fields{a.Identity == b.Identity, a.Exception == b.Exception, a.File == b.File, a.Line == b.Line, a.Message == b.Message, false}
}
func full(f Fields) bool { return f.Identity && f.Exception && f.File && f.Line && f.Message }
func Compare(remote, local, step, command string, localExit int, stepFailed bool) Match {
	r := Parse(remote, step, command, nil)
	l := Parse(local, step, command, &localExit)
	m := Match{Status: "INSUFFICIENT_EVIDENCE", Level: "STEP", Remote: r, Local: l, RemoteCount: r.FailureCount, LocalCount: l.FailureCount}
	if localExit == 0 {
		m.Status = "DIFFERENT_FAILURE"
		m.Reason = "the local job succeeded"
		return m
	}
	if !stepFailed {
		m.Status = "REPLAY_BLOCKED"
		m.Reason = "act did not fail in the original target step"
		return m
	}
	exitMatches := r.Exit != nil && *r.Exit == localExit
	m.MatchedFields.Exit = exitMatches
	for _, a := range r.Items {
		for _, b := range l.Items {
			if full(same(a, b)) {
				m.MatchedCount++
				break
			}
		}
	}
	if r.FailureCount > 0 && l.FailureCount > 0 && r.FailureCount != l.FailureCount {
		m.Status = "DIFFERENT_FAILURE"
		m.Reason = fmt.Sprintf("reported failure counts differ: remote %d, local %d; %d structured failures agree", r.FailureCount, l.FailureCount, m.MatchedCount)
		return m
	}

	if r.Level != "STEP" && l.Level != "STEP" {
		m.Level = r.Level
		m.MatchedFields = Fields{true, true, true, true, true, exitMatches}
		for _, a := range r.Items {
			found := false
			for _, b := range l.Items {
				if a.Identity != b.Identity {
					continue
				}
				f := same(a, b)
				m.MatchedFields.Exception = m.MatchedFields.Exception && f.Exception
				m.MatchedFields.File = m.MatchedFields.File && f.File
				m.MatchedFields.Line = m.MatchedFields.Line && f.Line
				m.MatchedFields.Message = m.MatchedFields.Message && f.Message

				found = true
				break
			}
			if !found {
				m.MatchedFields = Fields{Exit: exitMatches}
			}
		}
		if len(r.Items) != len(l.Items) {
			m.MatchedFields = Fields{Exit: exitMatches}
		}
		if m.MatchedCount == len(r.Items) && len(r.Items) == len(l.Items) && exitMatches {
			m.Status = "SAME_FAILURE"
			m.Reason = "all structured failure identities, exception/diagnostic types, source locations, messages and exit codes agree"
			return m
		}
		m.Status = "DIFFERENT_FAILURE"
		m.Reason = "structured evidence or exit code differs"
		return m
	}
	coarse := fingerprint.Compare(remote, local, localExit, stepFailed)
	if coarse.Score >= 0.8 && len(coarse.Remote.Lines) >= 2 && len(coarse.Local.Lines) >= 2 && exitMatches {
		m.Status = "LIKELY_MATCH"
		m.Reason = "step/error excerpt agrees; complete structured failure evidence is missing"
	} else {
		m.Reason = "not enough matching step evidence or remote exit code"
	}
	return m
}
