package resolver

import (
	"context"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"os"
	"strings"
	"testing"
)

const head = "1111111111111111111111111111111111111111"
const merge = "2222222222222222222222222222222222222222"

func fixture() *github.Bundle {
	w := `name: CI
on: [push, pull_request]
jobs:
  tests:
    name: test (${{ matrix.node }})
    runs-on: ubuntu-latest
    strategy:
      matrix:
        node: [20, 24]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node }}
      - name: Run tests
        run: npm test
`
	r := github.Run{ID: 123, RunAttempt: 2, HeadSHA: head, HeadBranch: "bug", Event: "pull_request", Path: ".github/workflows/ci.yml", Status: "completed", Conclusion: "failure", Repository: github.Repository{FullName: "o/r"}}
	log := "2026-09-03T13:21:08Z [command]/usr/bin/git log -1 --format=%H\n2026-09-03T13:21:08Z " + merge + "\n2026-09-03T13:21:19Z FAILED tests/auth.test.ts\n2026-09-03T13:21:19Z AssertionError: Expected: 200 Received: 500"
	return &github.Bundle{RunData: r, JobData: []github.Job{{ID: 7, Name: "test (24)", Conclusion: "failure", Labels: []string{"ubuntu-latest"}, Steps: []github.Step{{Name: "Run tests", Number: 4, Conclusion: "failure", StartedAt: "2026-09-03T13:21:19Z", CompletedAt: "2026-09-03T13:21:20Z"}}}}, Files: map[string]string{"o/r@" + head + ":.github/workflows/ci.yml": w, "o/r@" + merge + ":.github/workflows/ci.yml": w}, Logs: map[string]string{"/repos/o/r/actions/jobs/7/logs": log}}
}
func TestURLToReplayPlan(t *testing.T) {
	f := fixture()
	l, e := Resolve(context.Background(), f, "https://github.com/o/r/actions/runs/123", Options{})
	if e != nil {
		t.Fatal(e)
	}
	if l.Commit != merge || l.JobID != "tests" || fmt.Sprint(l.Matrix["node"]) != "24" || len(l.Blockers) > 0 {
		t.Fatalf("%+v", l)
	}
	path := t.TempDir() + "/runback.lock"
	if e = lockfile.Write(path, l); e != nil {
		t.Fatal(e)
	}
	again, e := lockfile.Read(path)
	if e != nil {
		t.Fatal(e)
	}
	p, e := replay.Prepare(again, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p.Workflow)
	if !strings.Contains(string(b), "node: 24") || strings.Contains(string(b), "node: 20") {
		t.Fatal(string(b))
	}
	if !strings.Contains(strings.Join(p.Args, " "), "--container-daemon-socket -") {
		t.Fatal(p.Args)
	}
}
func TestPRWithoutCheckoutEvidenceBlocked(t *testing.T) {
	f := fixture()
	f.Logs = nil
	l, e := Resolve(context.Background(), f, "https://github.com/o/r/actions/runs/123", Options{})
	if e != nil {
		t.Fatal(e)
	}
	if len(l.Blockers) == 0 {
		t.Fatal("missing PR SHA evidence must block")
	}
}
func TestPrivateRejected(t *testing.T) {
	f := fixture()
	f.RunData.Repository.Private = true
	if _, e := Resolve(context.Background(), f, "https://github.com/o/r/actions/runs/123", Options{}); e == nil {
		t.Fatal("private accepted")
	}
}
func TestSuccessRunRejected(t *testing.T) {
	f := fixture()
	f.RunData.Conclusion = "success"
	if _, e := Resolve(context.Background(), f, "https://github.com/o/r/actions/runs/123", Options{}); e == nil {
		t.Fatal("success accepted")
	}
}
