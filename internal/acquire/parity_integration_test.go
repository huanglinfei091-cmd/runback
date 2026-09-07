package acquire

import (
	"context"
	"encoding/json"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"os"
	"testing"
)

func TestLiveOnlineBundleCanonicalParity(t *testing.T) {
	path := os.Getenv("RUNBACK_PARITY_ONLINE_LOCK")
	if path == "" {
		t.Skip("requires a lock acquired from a real GitHub run")
	}
	l, e := lockfile.Read(path)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Resolve(context.Background(), l.URL, os.Getenv("RUNBACK_PARITY_BUNDLE"), "", false)
	if e != nil {
		t.Fatal(e)
	}
	onlineHash, bundleHash := SemanticHash(l), SemanticHash(b.Lock)
	status := "PASS"
	if onlineHash != bundleHash {
		status = "FAIL"
	}
	result := map[string]any{"status": status, "repository": l.Repository, "run_id": l.RunID, "run_attempt": l.Attempt, "online_semantic_hash": onlineHash, "bundle_semantic_hash": bundleHash, "online": Semantic(l), "bundle": Semantic(b.Lock)}
	data, _ := json.MarshalIndent(result, "", "  ")
	if out := os.Getenv("RUNBACK_PARITY_REPORT"); out != "" {
		if e = os.WriteFile(out, data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if status != "PASS" {
		t.Fatal(string(data))
	}
	t.Log(l.Repository, status, onlineHash)
}
