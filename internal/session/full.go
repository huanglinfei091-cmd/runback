package session

import (
	"context"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/replay"
	"github.com/huanglinfei091-cmd/runback/internal/runner"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Share only downloads. Every full execution still starts with a clean source checkout.
func cachedPlan(s State, p *replay.Plan) error {
	dir := filepath.Join(s.Cache, "uv")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	for i := range p.Args {
		if p.Args[i] == "--action-cache-path" {
			p.Args[i+1] = filepath.Join(s.Cache, "actions")
		}
		if p.Args[i] == "--container-options" && s.FullUVCachePath != "" {
			if strings.ContainsAny(dir+s.FullUVCachePath, ",\n\r") {
				return fmt.Errorf("invalid cache mount path")
			}
			p.Args[i+1] += " --mount " + strconv.Quote("type=bind,src="+dir+",dst="+s.FullUVCachePath)
		}
	}
	return nil
}
func Full(ctx context.Context, s State, out io.Writer) (runner.Result, error) {
	release, e := Acquire(s)
	if e != nil {
		return runner.Result{}, e
	}
	defer release()
	l := s.Lock
	l.Image = s.ImageID
	root := filepath.Join(s.Root, "full", fmt.Sprint(time.Now().UnixNano()))
	p, e := replay.Prepare(l, root)
	if e != nil {
		return runner.Result{}, e
	}
	if e = cachedPlan(s, &p); e != nil {
		return runner.Result{}, e
	}
	if _, e = run(ctx, s.Root, p.Root, "git", "clone", "--no-hardlinks", "--no-checkout", s.Original, p.Checkout); e != nil {
		return runner.Result{}, e
	}
	if _, e = run(ctx, s.Root, p.Root, "git", "-C", p.Checkout, "-c", "core.hooksPath=/dev/null", "checkout", "--detach", l.Commit); e != nil {
		return runner.Result{}, e
	}
	if _, e = run(ctx, s.Root, p.Root, "git", "-C", p.Checkout, "remote", "set-url", "origin", "https://github.com/"+l.Repository+".git"); e != nil {
		return runner.Result{}, e
	}
	r, e := runner.Execute(ctx, p, l, out)
	fmt.Fprintf(out, "Result: %s\nReport: %s\n", r.Status, filepath.Join(p.Root, "result.json"))
	return r, e
}
