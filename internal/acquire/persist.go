package acquire

import (
	"encoding/json"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"io"
	"os"
	"path/filepath"
)

func Persist(p replay.Plan, l lockfile.Lock, a Result, r runner.Result) error {
	files := map[string]any{
		"evidence.json":     map[string]any{"result": r, "semantic": Semantic(l), "semantic_hash": SemanticHash(l), "source": a.Source, "cache": a.CacheStatus, "cache_path": a.CachePath, "auth_mode": a.AuthMode},
		"replay-plan.json":  map[string]any{"workflow": p.Workflow, "checkout": p.Checkout, "args": p.Args, "image": l.Image, "network": l.Network, "network_source": l.NetworkSource},
		"run-evidence.json": a.Evidence,
	}
	for name, v := range files {
		b, e := json.MarshalIndent(v, "", "  ")
		if e != nil {
			return e
		}
		if e = lockfile.Atomic(filepath.Join(p.Root, name), b); e != nil {
			return e
		}
	}
	src, e := os.Open(filepath.Join(p.Root, "local.jsonl"))
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	defer src.Close()
	dst, e := os.Create(filepath.Join(p.Root, "act.log"))
	if e != nil {
		return e
	}
	_, e = io.Copy(dst, src)
	ce := dst.Close()
	if e != nil {
		return e
	}
	return ce
}
