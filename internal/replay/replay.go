package replay

import (
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/lockfile"
	"github.com/huanglinfei091-cmd/runback/internal/workflow"
	"os"
	"path/filepath"
)

type Plan struct {
	Root, Checkout, Workflow, Event, Empty, Home string
	Args                                         []string
}

func Prepare(l lockfile.Lock, base string) (Plan, error) {
	root, e := filepath.Abs(filepath.Join(base, l.Key()))
	if e != nil {
		return Plan{}, e
	}
	return prepareRoot(l, root)
}

// PrepareWorkspace uses an exact automatic workspace path. It avoids adding
// repository, run, job, or timestamp identity to the physical path.
func PrepareWorkspace(l lockfile.Lock, root string) (Plan, error) {
	root, e := filepath.Abs(root)
	if e != nil {
		return Plan{}, e
	}
	return prepareRoot(l, root)
}

func prepareRoot(l lockfile.Lock, root string) (Plan, error) {
	var p Plan
	if e := l.Validate(); e != nil {
		return p, e
	}
	p = Plan{Root: root, Checkout: filepath.Join(root, "workspace"), Workflow: filepath.Join(root, "replay.yml"), Event: filepath.Join(root, "event.json"), Empty: filepath.Join(root, "empty.env"), Home: filepath.Join(root, "home")}
	w, e := workflow.Parse([]byte(l.Workflow))
	if e != nil {
		return p, e
	}
	s, e := workflow.Select(w, l.JobName)
	if e != nil {
		return p, e
	}
	if s.ID != l.JobID {
		return p, fmt.Errorf("lock job differs from workflow")
	}
	workflow.PinObserved(&w, l.JobID, l.ObservedVersions)
	b, e := workflow.Replay(w, s)
	if e != nil {
		return p, e
	}
	for _, dir := range []string{root, p.Home, filepath.Join(p.Home, ".config")} {
		if e = os.MkdirAll(dir, 0700); e != nil {
			return p, e
		}
	}
	for path, data := range map[string][]byte{p.Workflow: b, p.Event: l.EventJSON, p.Empty: {}} {
		if e = lockfile.Atomic(path, data); e != nil {
			return p, e
		}
	}
	network := l.Network
	if network == "" {
		network = "bridge"
	}
	p.Args = []string{l.Event, "-C", p.Checkout, "-W", p.Workflow, "-j", l.JobID, "-e", p.Event, "-P", l.Runner + "=" + l.Image, "--actor", l.Actor, "--container-architecture", "linux/amd64", "--container-daemon-socket", "-", "--network", network, "--secret-file", p.Empty, "--var-file", p.Empty, "--env-file", p.Empty, "--input-file", p.Empty, "--action-cache-path", filepath.Join(root, "actions"), "--pull=false", "--use-new-action-cache", "--container-options", "--label io.runback.case=" + l.Key(), "--json"}
	return p, nil
}
