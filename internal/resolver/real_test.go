package resolver

import (
	"context"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"os"
	"strings"
	"testing"
)

func TestRealClickRegression(t *testing.T) {
	f, e := github.ReadBundle("../../testdata/real/click.json")
	if e != nil {
		t.Fatal(e)
	}
	l, e := Resolve(context.Background(), f, f.URL, Options{})
	if e != nil {
		t.Fatal(e)
	}
	if l.Commit != "4a0598c3c179b70b5116a800b485571bcc341ddf" || l.Matrix["python"] != "3.13" || l.ObservedVersions["python-version"] != "3.13.15" || len(l.Blockers) > 0 {
		t.Fatalf("%+v", l)
	}
	p, e := replay.Prepare(l, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p.Workflow)
	if !strings.Contains(string(b), "python-version: 3.13.15") {
		t.Fatal(string(b))
	}
}
