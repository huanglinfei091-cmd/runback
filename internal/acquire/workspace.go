package acquire

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// AllocateWorkspace returns an identity-free transient path. OnlineSource and
// BundleSource both reach this allocator through the shared CLI path.
func AllocateWorkspace() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("automatic transient workspace is supported on Linux")
	}
	base := filepath.Join(os.TempDir(), "rb")
	if e := os.MkdirAll(base, 0700); e != nil {
		return "", e
	}
	for i := 0; i < 16; i++ {
		b := make([]byte, 6)
		if _, e := rand.Read(b); e != nil {
			return "", e
		}
		root := filepath.Join(base, hex.EncodeToString(b))
		if e := os.Mkdir(root, 0700); e == nil {
			return root, nil
		} else if !os.IsExist(e) {
			return "", e
		}
	}
	return "", fmt.Errorf("cannot allocate a unique RunBack workspace")
}
