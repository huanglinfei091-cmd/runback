//go:build linux

package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeDocker(t *testing.T, body string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n" +
		"if [ -n \"$RUNBACK_GITHUB_TOKEN$GH_TOKEN$GITHUB_TOKEN\" ]; then exit 90; fi\n" +
		"printf '%s\\n' \"$*\" >> " + log + "\n" + body
	if e := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", dir)
	return dir, log
}

func TestHealthyDefaultDoesNotUseFallback(t *testing.T) {
	_, log := fakeDocker(t, "exit 0\n")
	t.Setenv("RUNBACK_GITHUB_TOKEN", "must-not-leak")
	name, source, e := SelectNetwork(context.Background(), "runner-image")
	if e != nil || name != "" || source != "GENERIC_DEFAULT" {
		t.Fatal(name, source, e)
	}
	b, _ := os.ReadFile(log)
	if strings.Contains(string(b), "network create") || strings.Contains(string(b), "must-not-leak") {
		t.Fatal(string(b))
	}
}

func TestUnhealthyDefaultReusesOneManagedFallback(t *testing.T) {
	_, log := fakeDocker(t, `
if [ "$1" = "run" ]; then
  case "$*" in *"--network bridge"*) exit 1;; *"--network runback-replay"*) exit 0;; esac
fi
if [ "$1 $2" = "network ls" ]; then echo runback-replay; exit 0; fi
if [ "$1 $2 $3" = "network inspect runback-replay" ]; then echo 'bridge|false|true'; exit 0; fi
exit 2
`)
	t.Setenv("GH_TOKEN", "must-not-leak")
	name, source, e := SelectNetwork(context.Background(), "runner-image")
	if e != nil || name != "runback-replay" || source != "MANAGED_FALLBACK" {
		t.Fatal(name, source, e)
	}
	b, _ := os.ReadFile(log)
	calls := string(b)
	if strings.Contains(calls, "network create") || strings.Contains(calls, "must-not-leak") || strings.Contains(calls, "werkzeug") || strings.Contains(calls, "pallets") {
		t.Fatal(calls)
	}
}

func TestUnhealthyDefaultCreatesOneManagedFallbackWithoutIPAM(t *testing.T) {
	_, log := fakeDocker(t, `
if [ "$1" = "run" ]; then
  case "$*" in *"--network bridge"*) exit 1;; *"--network runback-managed"*) exit 0;; esac
fi
if [ "$1 $2" = "network ls" ]; then exit 0; fi
if [ "$1 $2" = "network create" ]; then echo created-id; exit 0; fi
exit 2
`)
	name, source, e := SelectNetwork(context.Background(), "runner-image")
	if e != nil || name != managedNetwork || source != "MANAGED_FALLBACK" {
		t.Fatal(name, source, e)
	}
	b, _ := os.ReadFile(log)
	calls := string(b)
	if strings.Count(calls, "network create") != 1 || strings.Contains(calls, "subnet") || strings.Contains(calls, "gateway") {
		t.Fatal(calls)
	}
}

func TestUnhealthyManagedNetworkIsNotReused(t *testing.T) {
	_, log := fakeDocker(t, `
if [ "$1" = "run" ]; then
  case "$*" in *"--network runback-managed"*) exit 0;; *) exit 1;; esac
fi
if [ "$1 $2" = "network ls" ]; then echo runback-replay; exit 0; fi
if [ "$1 $2 $3" = "network inspect runback-replay" ]; then echo 'bridge|false|true'; exit 0; fi
if [ "$1 $2" = "network create" ]; then echo created-id; exit 0; fi
exit 2
`)
	name, source, err := SelectNetwork(context.Background(), "runner-image")
	if err != nil || name != managedNetwork || source != "MANAGED_FALLBACK" {
		t.Fatal(name, source, err)
	}
	b, _ := os.ReadFile(log)
	calls := string(b)
	if !strings.Contains(calls, "--network runback-replay") || strings.Count(calls, "network create") != 1 || strings.Contains(calls, "subnet") || strings.Contains(calls, "gateway") {
		t.Fatal(calls)
	}
}
