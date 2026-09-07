package workflow

import (
	"strings"
	"testing"
)

func TestMatrixIncludeExclude(t *testing.T) {
	w, e := Parse([]byte(`name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        node: [20, 22]
        os: [ubuntu-latest]
        exclude:
          - node: 20
        include:
          - node: 22
            os: ubuntu-latest
            mode: strict
          - node: 24
            os: ubuntu-latest
    steps:
      - run: npm test
`))
	if e != nil {
		t.Fatal(e)
	}
	rows, e := Matrix(w.Jobs["test"].Strategy.Matrix)
	if e != nil || len(rows) != 2 || rows[0]["mode"] != "strict" {
		t.Fatalf("%v %v", rows, e)
	}
	s, e := Select(w, "test (24, ubuntu-latest)")
	if e != nil || s.Matrix["node"] != 24 {
		t.Fatalf("%v %v", s, e)
	}
	b, e := Replay(w, s)
	if e != nil {
		t.Fatal(e)
	}
	w2, e := Parse(b)
	if e != nil {
		t.Fatal(e)
	}
	m, e := Matrix(w2.Jobs["test"].Strategy.Matrix)
	if e != nil || len(m) != 1 {
		t.Fatalf("%v %v", m, e)
	}
}
func TestCustomIncludeName(t *testing.T) {
	w, e := Parse([]byte(`jobs:
  test:
    name: ${{ matrix.name || matrix.python }}
    runs-on: ${{ matrix.os || 'ubuntu-latest' }}
    strategy:
      matrix:
        include:
          - python: '3.13'
          - python: '3.14'
            name: Windows
            os: windows-latest
    steps:
      - run: pytest
`))
	if e != nil {
		t.Fatal(e)
	}
	s, e := Select(w, "3.13")
	if e != nil || s.Runner != "ubuntu-latest" || s.Matrix["python"] != "3.13" {
		t.Fatalf("%+v %v", s, e)
	}
}
func TestAmbiguityFails(t *testing.T) {
	w, _ := Parse([]byte("jobs:\n  test:\n    name: test\n    strategy:\n      matrix:\n        node: [20, 24]\n"))
	if _, e := Select(w, "test"); e == nil {
		t.Fatal("ambiguous name accepted")
	}
}
func TestDynamicFails(t *testing.T) {
	if _, e := Matrix("${{ fromJSON(needs.prepare.outputs.matrix) }}"); e == nil {
		t.Fatal("dynamic matrix accepted")
	}
}
func TestUnsupportedExpression(t *testing.T) {
	s, ok := Render("${{ github.ref }}", nil)
	if ok || !strings.Contains(s, "github.ref") {
		t.Fatal(s, ok)
	}
}
func TestNoStepNumberGuess(t *testing.T) {
	j := Job{Steps: []Step{{Name: "test", Run: "a"}, {Name: "test", Run: "b"}}}
	if _, _, e := FailedCommand(j, "test", nil); e == nil {
		t.Fatal("duplicate step accepted")
	}
}
func TestMatrixOriginalIncludeRules(t *testing.T) {
	raw := map[string]any{"fruit": []any{"apple", "pear"}, "include": []any{map[string]any{"color": "green"}, map[string]any{"fruit": "apple", "color": "pink"}, map[string]any{"fruit": "banana"}, map[string]any{"fruit": "banana", "color": "yellow"}}}
	rows, e := Matrix(raw)
	if e != nil || len(rows) != 4 || rows[0]["color"] != "pink" {
		t.Fatalf("%v %v", rows, e)
	}
}
