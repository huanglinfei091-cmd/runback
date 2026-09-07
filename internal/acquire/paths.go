package acquire

import (
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"os"
	"path/filepath"
	"strings"
)

func SameIdentity(a, b lockfile.Lock) bool {
	return strings.EqualFold(a.Repository, b.Repository) && a.RunID == b.RunID && a.Attempt == b.Attempt && a.Commit == b.Commit && a.JobNumber == b.JobNumber && a.JobID == b.JobID
}
func ExistingLock(path string, want lockfile.Lock) (*lockfile.Lock, error) {
	info, e := os.Lstat(path)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("LOCK_IDENTITY_MISMATCH: lock is not a regular file")
	}
	old, e := lockfile.Read(path)
	if e != nil || !SameIdentity(old, want) {
		return nil, fmt.Errorf("LOCK_IDENTITY_MISMATCH: repository/run/attempt/commit/job differ; existing lock preserved")
	}
	return &old, nil
}
func SafeWorkdir(base string, l lockfile.Lock) error {
	root, e := filepath.Abs(filepath.Join(base, l.Key()))
	if e != nil {
		return e
	}
	return safeWorkspace(root, l)
}

// SafeWorkspace validates an exact RunBack-managed workspace root. It is used
// by the automatic short Linux allocator; explicit --work-dir keeps the
// historical base/<lock-key> layout through SafeWorkdir.
func SafeWorkspace(root string, l lockfile.Lock) error {
	root, e := filepath.Abs(root)
	if e != nil {
		return e
	}
	return safeWorkspace(root, l)
}

func safeWorkspace(root string, l lockfile.Lock) error {
	// Never follow a user-supplied symlink into a different checkout.
	for dir := root; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		info, e := os.Lstat(dir)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("WORKDIR_NOT_SAFE: symlink in replay path")
		}
	}
	entries, e := os.ReadDir(root)
	if e == nil && len(entries) > 0 {
		b, e := os.ReadFile(filepath.Join(root, ".runback-owned.json"))
		var old lockfile.Lock
		if e != nil || json.Unmarshal(b, &old) != nil || !SameIdentity(old, l) {
			return fmt.Errorf("WORKDIR_NOT_SAFE: existing non-managed replay directory preserved")
		}
	}
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return e
	}
	b, _ := json.Marshal(l)
	// Identity marker contains no raw log or workflow; only the required identity.
	var identity map[string]any
	json.Unmarshal(b, &identity)
	for k := range identity {
		switch k {
		case "repository", "run_id", "attempt", "commit", "job_number", "job_id":
		default:
			delete(identity, k)
		}
	}
	b, _ = json.Marshal(identity)
	return lockfile.Atomic(filepath.Join(root, ".runback-owned.json"), b)
}
