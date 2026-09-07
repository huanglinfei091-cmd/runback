//go:build linux

package session

import (
	"context"
	"strings"
	"testing"
)

func TestDevAndDockerSubprocessIsolation(t *testing.T) {
	for _, key := range []string{"GH_TOKEN", "RUNBACK_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(key, "session-process-secret")
	}
	root := t.TempDir()
	out, e := output(context.Background(), root, root, "sh", "-c", "env")
	if e != nil || strings.Contains(out, "session-process-secret") || strings.Contains(out, "GH_TOKEN=") {
		t.Fatal("session subprocess credential leak", e)
	}
	s := State{ID: "test", Root: root, Worktree: root, Cache: root, ImageID: "sha256:test"}
	for _, interactive := range []bool{true, false} {
		args, e := DockerArgs(s, "probe", interactive, "true")
		if e != nil {
			t.Fatal(e)
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "GH_TOKEN") || strings.Contains(joined, "RUNBACK_GITHUB_TOKEN") || strings.Contains(joined, "session-process-secret") {
			t.Fatal("docker/dev arguments contain credentials")
		}
	}
}
