//go:build linux

package acquire

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestAutomaticWorkspaceIsShortAndIdentityFree(t *testing.T) {
	root, e := AllocateWorkspace()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = os.Remove(root) })
	if !regexp.MustCompile(`^/tmp/rb/[a-f0-9]{12}$`).MatchString(root) {
		t.Fatalf("unexpected automatic workspace %q", root)
	}
	for _, identity := range []string{"pallets", "werkzeug", "32448268750", "96671790465"} {
		if strings.Contains(root, identity) {
			t.Fatalf("identity %q leaked into path %q", identity, root)
		}
	}
	info, e := os.Stat(root)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("workspace permissions: %v %v", info, e)
	}
}
