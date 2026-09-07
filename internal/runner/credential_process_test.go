//go:build linux

package runner

import (
	"context"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionSubprocessesReceiveSanitizedEnvironment(t *testing.T) {
	for _, key := range []string{"GH_TOKEN", "RUNBACK_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(key, "process-probe-secret")
	}
	home := t.TempDir()
	p := replay.Plan{Root: home, Home: home}
	out, e := command(context.Background(), p, "sh", "-c", "env").CombinedOutput()
	if e != nil || strings.Contains(string(out), "process-probe-secret") || strings.Contains(string(out), "GH_TOKEN=") || strings.Contains(string(out), "RUNBACK_GITHUB_TOKEN=") {
		t.Fatal("execution subprocess credential leak", e)
	}
}
func TestDoctorDockerProcessDoesNotReceiveToken(t *testing.T) {
	for _, key := range []string{"GH_TOKEN", "RUNBACK_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(key, "doctor-probe-secret")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nif [ -n \"$RUNBACK_GITHUB_TOKEN$GH_TOKEN$GITHUB_TOKEN\" ]; then echo CREDENTIAL_LEAK; exit 9; fi\necho Linux\n"
	for _, name := range []string{"git", "act", "docker"} {
		os.WriteFile(filepath.Join(bin, name), []byte(script), 0700)
	}
	t.Setenv("PATH", bin)
	var out strings.Builder
	if !Doctor(context.Background(), &out) || strings.Contains(out.String(), "CREDENTIAL_LEAK") {
		t.Fatal("doctor process inherited acquisition credential")
	}
}
