package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/acquire"
	"github.com/huanglinfei091-cmd/runback/internal/doctor"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"github.com/huanglinfei091-cmd/runback/internal/session"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(mainRun(ctx, os.Args[1:]))
}
func mainRun(ctx context.Context, args []string) int {
	started := time.Now()
	defer func() {
		fmt.Fprintf(os.Stderr, "CLI elapsed: %.3fs (through final result)\n", time.Since(started).Seconds())
	}()
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		fmt.Println("RunBack — reproduce a specific failed GitHub Actions run\n\nrunback <URL> [--bundle evidence.json] [--job NAME] [--dry-run]\nrunback inspect <URL> [--bundle evidence.json]\nrunback replay [--dry-run]\nrunback dev [--session ID]\nrunback replay --step [NAME] [--session ID]\nrunback verify [--session ID]\nrunback reproduce --session ID\nrunback shell\nrunback doctor\nrunback version\n\nOptions: --refresh --job NAME --timeout 20m --image IMAGE --network NETWORK --json\nOptional filesystem overrides: --lock PATH --work-dir PATH\nURL mode automatically manages evidence, lock and replay workspace.")
		return 0
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Println(versionOutput(version))
		return 0
	}
	if args[0] == "doctor" {
		if doctor.Run(ctx, os.Stdout) {
			return 0
		}
		return 2
	}
	action := "run"
	if args[0] == "inspect" || args[0] == "replay" || args[0] == "shell" || args[0] == "dev" || args[0] == "verify" || args[0] == "reproduce" {
		action = args[0]
		args = args[1:]
	}
	f := flag.NewFlagSet("runback", flag.ContinueOnError)
	lock := f.String("lock", "runback.lock", "lockfile")
	base := f.String("work-dir", ".runback", "replay workspaces")
	refresh := f.Bool("refresh", false, "reacquire all online evidence")
	bundle := f.String("bundle", "", "copied public evidence")
	network := f.String("network", "", "Docker network override (recorded in lock)")
	image := f.String("image", "", "explicit act runner image override (recorded in lock)")
	sessionID := f.String("session", "", "session ID; defaults to active session")
	step := f.String("step", "", "fast replay selected failed step")
	job := f.String("job", "", "remote job name or number")
	dry := f.Bool("dry-run", false, "prepare configuration without execution")
	asJSON := f.Bool("json", false, "print lock as JSON")
	timeout := f.Duration("timeout", 20*time.Minute, "maximum replay time")
	// Permit the product's URL-first syntax with flags after the URL.
	var flags, pos []string
	valueFlags := map[string]bool{"--lock": true, "--work-dir": true, "--bundle": true, "--job": true, "--timeout": true, "--image": true, "--session": true, "--network": true, "--step": true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--step" && (i+1 == len(args) || strings.HasPrefix(args[i+1], "-")) {
			flags = append(flags, "--step=failed")
			continue
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if valueFlags[a] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			pos = append(pos, a)
		}
	}
	if e := f.Parse(flags); e != nil {
		return 2
	}
	explicit := map[string]bool{}
	f.Visit(func(v *flag.Flag) { explicit[v.Name] = true })
	baseSessions, e := session.Base()
	if e != nil {
		return fail(e)
	}
	if action == "dev" || action == "verify" || (action == "replay" && *step != "") || (action == "reproduce" && len(pos) == 0) {
		if len(pos) > 0 {
			return fail(fmt.Errorf("unexpected positional argument"))
		}
		s, e := session.Load(baseSessions, *sessionID)
		if e != nil {
			return fail(e)
		}
		if *network != "" && *network != s.Lock.Network {
			return fail(fmt.Errorf("session uses recorded network %s; reproduce with --network to create another session", s.Lock.Network))
		}
		if *image != "" {
			return fail(fmt.Errorf("reproduce with --image to create a session with a different verified image"))
		}
		ctx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()
		if action == "verify" {
			r, e := session.Verify(ctx, s, os.Stdout)
			if e != nil {
				return fail(e)
			}
			if r.Status == "FULL_JOB_PASSED" {
				return 0
			}
			return 1
		}
		if action == "reproduce" {
			r, e := session.Full(ctx, s, os.Stdout)
			if e != nil {
				return fail(e)
			}
			if r.Status == "SAME_FAILURE" {
				return 0
			}
			return 1
		}
		if *step != "" && *step != "failed" && *step != s.Lock.Failure.Name && *step != strings.TrimPrefix(s.Lock.Failure.Name, "Run ") {
			return fail(fmt.Errorf("M2 replays only the recorded failed step: %s", s.Lock.Failure.Name))
		}
		r, e := session.Debug(ctx, &s, action == "dev", os.Stdout)
		if e != nil {
			return fail(e)
		}
		if r.Status == "SAME_FAILURE" || r.Status == "STEP_PASSED_UNVERIFIED" || r.Status == "DEV_EXITED" {
			return 0
		}
		return 1
	}
	if action == "reproduce" {
		action = "run"
	}
	var l lockfile.Lock
	var acquisition acquire.Result
	autoWorkspace := false
	imageOrigin, networkOrigin := "GENERIC_DEFAULT", "GENERIC_DEFAULT"
	if action == "run" || action == "inspect" {
		if len(pos) != 1 {
			fmt.Fprintln(os.Stderr, "provide one failed run URL")
			return 2
		}

		acquisition, e = acquire.Resolve(ctx, pos[0], *bundle, *job, *refresh)
		if e != nil {
			return fail(e)
		}
		l = acquisition.Lock
		fmt.Printf("Evidence: %s\nCache: %s\nAuth mode: %s\n", acquisition.Source, acquisition.CacheStatus, acquisition.AuthMode)
		if explicit["lock"] {
			old, err := acquire.ExistingLock(*lock, l)
			if err != nil {
				return fail(err)
			}
			if old != nil {
				l.Image = old.Image
				l.Network = old.Network
				imageOrigin = "RECORDED_LOCK"
				networkOrigin = "RECORDED_LOCK"
			}
		}
		if imageOrigin == "GENERIC_DEFAULT" {
			if old, err := session.Load(baseSessions, ""); err == nil && acquire.SameIdentity(old.Lock, l) {
				l.Image = old.ImageOverride
				l.Network = old.Lock.Network
				imageOrigin = "RECORDED_SESSION"
				networkOrigin = "RECORDED_SESSION"
			}
		}
		if !explicit["work-dir"] {
			path, err := acquire.AllocateWorkspace()
			if err != nil {
				return fail(err)
			}
			*base = path
			autoWorkspace = true
		}
		if !explicit["lock"] {
			if autoWorkspace {
				*lock = filepath.Join(*base, "runback.lock")
			} else {
				*lock = filepath.Join(*base, l.Key(), "runback.lock")
			}
		}

	} else {
		if len(pos) > 0 {
			return fail(fmt.Errorf("unexpected positional argument"))
		}
		l, e = lockfile.Read(*lock)
		if e != nil {
			return fail(e)
		}
	}

	if *network != "" {
		l.Network = *network
		networkOrigin = "CLI_OVERRIDE"
	}
	if *image != "" {
		l.Image = *image
		imageOrigin = "CLI_OVERRIDE"
	}
	if action == "run" && networkOrigin == "GENERIC_DEFAULT" && l.Network == "" {
		selected, source, err := runner.SelectNetwork(ctx, l.Image)
		if err != nil {
			return fail(err)
		}
		l.Network = selected
		networkOrigin = source
	}
	l.NetworkSource = networkOrigin
	if action == "run" || action == "inspect" {
		if autoWorkspace {
			e = acquire.SafeWorkspace(*base, l)
		} else {
			e = acquire.SafeWorkdir(*base, l)
		}
		if e != nil {
			return fail(e)
		}
	}
	if action != "shell" {
		if e = lockfile.Write(*lock, l); e != nil {
			return fail(e)
		}
	}
	networkName := l.Network
	if networkName == "" {
		networkName = "bridge"
	}
	fmt.Printf("Image: %s (%s)\nNetwork: %s\nNetwork Source: %s\n", l.Image, imageOrigin, networkName, networkOrigin)
	if *asJSON {
		b, _ := json.MarshalIndent(l, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("Workflow      %s\nCommit        %s\nJob           %s [%s]\nMatrix        %v\nRunner        %s\nFailed step   %s\nCommand       %s\nLock          %s\n", l.WorkflowPath, l.Commit, l.JobName, l.JobID, l.Matrix, l.Runner, l.Failure.Name, l.Failure.Command, *lock)
		for _, s := range l.Warnings {
			fmt.Println("NOTE         ", s)
		}
		for _, s := range l.Blockers {
			fmt.Println("BLOCKED      ", s)
		}
	}
	if action == "inspect" {
		return 0
	}
	var p replay.Plan
	if autoWorkspace {
		p, e = replay.PrepareWorkspace(l, *base)
	} else {
		p, e = replay.Prepare(l, *base)
	}
	if e != nil {
		return fail(e)
	}
	if *dry {
		fmt.Println("Prepared:", p.Workflow)
		b, _ := json.Marshal(append([]string{"act"}, p.Args...))
		fmt.Println("Command argv:", string(b))
		return 0
	}
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	if action == "shell" {
		return fail(runner.Shell(ctx, p))
	}
	r, e := runner.Execute(ctx, p, l, os.Stdout)
	if pe := acquire.Persist(p, l, acquisition, r); pe != nil {
		fmt.Fprintln(os.Stderr, "EVIDENCE_SAVE_FAILED:", pe)
	}

	fmt.Printf("\nResult: %s\nStage: %s\nCause: %s\nEvidence level: %s\nRemote failures: %d\nLocal failures: %d\nMatched: %d\n%s\nReport: %s\n", r.Status, r.Stage, r.Cause, r.Evidence.Level, r.Evidence.RemoteCount, r.Evidence.LocalCount, r.Evidence.MatchedCount, r.Evidence.Reason, filepath.Join(p.Root, "result.json"))
	evidenceJSON, _ := json.MarshalIndent(r.Evidence, "", "  ")
	fmt.Println(string(evidenceJSON))
	if e != nil {
		return fail(e)
	}
	return completeReproduction(ctx, os.Stdout, baseSessions, l, p, r, session.Create, session.StopReproductionContainers)
}

func versionOutput(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "dev" {
		return "RunBack dev"
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return "RunBack " + value
}
func fail(e error) int {
	if e == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, "RunBack:", e)
	return 2
}
