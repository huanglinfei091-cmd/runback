// Package failure distinguishes step similarity from structured failure identity.
package failure

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/huanglinfei091-cmd/runback/internal/fingerprint"
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
var typescript = regexp.MustCompile(`^(.+?\.(?:[cm]?tsx?|[cm]?jsx?))\(([1-9]\d*),([1-9]\d*)\): error (TS\d+): (.+)$`)
var typescriptError = regexp.MustCompile(`\berror TS\d+:`)
var logTimestamp = regexp.MustCompile(`^\d{4}-\d\d-\d\dT[0-9:.]+Z ?`)
var logANSI = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
var pyright = regexp.MustCompile(`^(.+?\.pyi?):(\d+):(\d+)\s+-\s+error:\s+(.+?)(?:\s+\(([A-Za-z][A-Za-z0-9_-]+)\))?$`)
var diagnosticCode = regexp.MustCompile(`^\[([a-zA-Z0-9_-]+)\]$`)
var frame = regexp.MustCompile(`^(.+?\.py):(\d+): in ([A-Za-z0-9_]+)$`)
var exception = regexp.MustCompile(`^E\s+((?:[A-Za-z0-9_.]*(?:Error|Exception))|Failed)(?::\s*(.*))?$`)
var goTestFailure = regexp.MustCompile(`^--- FAIL: ([^\s(]+) \([^)]+\)$`)
var goTestDiagnostic = regexp.MustCompile(`^(.+?_test\.go):(\d+):\s+(.+)$`)
var remoteExit = regexp.MustCompile(`Process completed with exit code (\d+)`)
var uuid = regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
var temp = regexp.MustCompile(`(?:/tmp|/home/runner/work/_temp)/[^ /:]+`)
var vitestHeader = regexp.MustCompile(`^FAIL\s+(.+?\.(?:[cm]?[jt]sx?))\s+>\s+(.+)$`)
var vitestLikeHeader = regexp.MustCompile(`^FAIL\s+(.+?)\s+>\s+(.+)$`)
var vitestException = regexp.MustCompile(`^((?:[A-Za-z_$][A-Za-z0-9_.$]*)?(?:Error|Exception)):\s*(.+)$`)
var vitestFrame = regexp.MustCompile(`^❯\s+(?:.+\s+)?(\S+\.(?:[cm]?[jt]sx?)):(\d+):(\d+)$`)

func Normalize(s string) string {
	s = fingerprint.Normalize(s)
	s = uuid.ReplaceAllString(s, "<uuid>")
	s = temp.ReplaceAllString(s, "<temp>")
	return strings.TrimSpace(s)
}

type vitestFailure struct {
	item   Item
	name   string
	column int
	marker string
}

func cleanVitestPath(s string) string {
	s = strings.ReplaceAll(s, `\`, "/")
	return strings.TrimPrefix(s, "./")
}

// parseVitest requires a complete FAIL block. Summary totals and annotation
// echoes are deliberately insufficient because they cannot establish identity.
func parseVitest(log string) ([]Item, []string, int) {
	var failures []vitestFailure
	var active []int
	for _, rawLine := range strings.Split(log, "\n") {
		line := Normalize(logTimestamp.ReplaceAllString(logANSI.ReplaceAllString(rawLine, ""), ""))
		if m := vitestHeader.FindStringSubmatch(line); m != nil {
			file := cleanVitestPath(m[1])
			name := strings.TrimSpace(m[2])
			// Vitest may print several parameterized FAIL headers followed by one
			// shared error body. Group only consecutive headers for the same file
			// before any body evidence has been observed.
			if len(active) > 0 {
				first := failures[active[0]]
				if first.item.File != file || first.item.Exception != "" || first.item.Line != 0 {
					active = nil
				}
			}
			failures = append(failures, vitestFailure{
				item:   Item{Kind: "vitest", File: file},
				name:   name,
				marker: "vitest: " + file + " > " + name,
			})
			active = append(active, len(failures)-1)
			continue
		}
		if m := vitestLikeHeader.FindStringSubmatch(line); m != nil {
			file := cleanVitestPath(m[1])
			name := strings.TrimSpace(m[2])
			failures = append(failures, vitestFailure{name: name, marker: "vitest: " + file + " > " + name})
			active = nil
			continue
		}
		if len(active) == 0 {
			continue
		}
		if strings.HasPrefix(line, "⎯") || strings.HasPrefix(line, "Test Files") || strings.HasPrefix(line, "Tests ") || strings.Contains(line, "Unhandled Error") {
			active = nil
			continue
		}
		if m := vitestException.FindStringSubmatch(line); m != nil && failures[active[0]].item.Exception == "" {
			for _, index := range active {
				failures[index].item.Exception = m[1]
				failures[index].item.Message = Normalize(m[2])
			}
			continue
		}
		if (strings.HasPrefix(line, "Expected:") || strings.HasPrefix(line, "Received:")) && failures[active[0]].item.Exception != "" {
			for _, index := range active {
				failures[index].item.Message += "\n" + line
			}
			continue
		}
		if m := vitestFrame.FindStringSubmatch(line); m != nil {
			file := cleanVitestPath(m[1])
			for _, index := range active {
				failure := &failures[index]
				if file == failure.item.File && failure.item.Line == 0 {
					failure.item.Line, _ = strconv.Atoi(m[2])
					failure.column, _ = strconv.Atoi(m[3])
				}
			}
		}
	}

	identities := map[string]int{}
	for i := range failures {
		f := &failures[i]
		if f.item.File == "" || f.name == "" || f.item.Line == 0 || f.column == 0 || f.item.Exception == "" || f.item.Message == "" {
			continue
		}
		f.item.Identity = fmt.Sprintf("vitest:%s:%d:%d:%s", f.item.File, f.item.Line, f.column, f.name)
		identities[f.item.Identity]++
	}

	items := []Item{}
	unparsed := []string{}
	for _, f := range failures {
		if f.item.Identity == "" || identities[f.item.Identity] != 1 {
			unparsed = append(unparsed, f.marker)
			continue
		}
		items = append(items, f.item)
	}
	return items, unparsed, len(failures)
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
	activeTS := -1
	for _, line := range lines {
		raw := logTimestamp.ReplaceAllString(logANSI.ReplaceAllString(line, ""), "")
		line = Normalize(line)
		if m := typescript.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[2])
			diagnostics = append(diagnostics, Item{Kind: "typescript", File: m[1], Line: n,
				Exception: m[4], Message: m[5], Identity: fmt.Sprintf("%s:%s:%s:%s", m[1], m[2], m[3], m[4])})
			activeTS = len(diagnostics) - 1
			continue
		}
		if typescriptError.MatchString(line) {
			// Count unsupported formats too, so a partial parse cannot become SAME_FAILURE.
			diagnostics = append(diagnostics, Item{Kind: "typescript", Message: line})
			activeTS = -1
			continue
		}
		if activeTS >= 0 && line != "" && (strings.HasPrefix(raw, "  ") || strings.HasPrefix(raw, "\t")) {
			diagnostics[activeTS].Message += "\n" + line
			continue
		}
		activeTS = -1
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
	vitestItems, vitestUnparsed, vitestCount := parseVitest(log)
	e.FailureCount = len(diagnostics) + len(nodes) + len(goFailures) + vitestCount
	for _, d := range diagnostics {
		if d.Exception != "" {
			if d.Kind == "mypy" {
				d.Exception = "mypy[" + d.Exception + "]"
			}
			if d.Identity == "" {
				d.Identity = fmt.Sprintf("%s:%d:%s", d.File, d.Line, d.Exception)
			}
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
	e.Items = append(e.Items, vitestItems...)
	e.UnparsedFailures = append(e.UnparsedFailures, vitestUnparsed...)
	if len(e.Items) > 0 && len(e.Items) == e.FailureCount {
		e.Level = "TEST"
		if len(nodes) == 0 && len(goFailures) == 0 && vitestCount == 0 {
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
