package online

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"github.com/huanglinfei091-cmd/runback/internal/resolver"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	t              *testing.T
	bundle         *github.Bundle
	api, log       *httptest.Server
	counts         map[string]int
	workflowStatus int
	remaining      string
	attempt        int
	pages          bool
	auth           string
	gzip           bool
	blockedRef     string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	b, e := github.ReadBundle("../../docs/m1/evidence/case-a-bundle.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &fixture{t: t, bundle: b, counts: map[string]int{}}
	f.log = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, key := range []string{"Authorization", "Cookie", "RUNBACK_GITHUB_TOKEN", "GH_TOKEN"} {
			if r.Header.Get(key) != "" {
				t.Errorf("download leaked %s", key)
			}
		}
		f.counts["download"]++
		for _, text := range b.Logs {
			if f.gzip {
				w.Header().Set("Content-Encoding", "gzip")
				z := gzip.NewWriter(w)
				io.WriteString(z, text)
				z.Close()
			} else {
				io.WriteString(w, text)
			}
			break
		}
	}))
	f.api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != f.auth {
			t.Errorf("API auth=%q want=%q", r.Header.Get("Authorization"), f.auth)
		}
		countKey := r.URL.Path
		if strings.Contains(countKey, "/contents/") {
			countKey = r.URL.RequestURI()
		}
		f.counts[countKey]++
		enc := json.NewEncoder(w)
		switch {
		case strings.Contains(r.URL.Path, "/contents/"):
			if f.blockedRef != "" && r.URL.Query().Get("ref") == f.blockedRef {
				w.WriteHeader(404)
				return
			}
			if f.workflowStatus > 0 {
				if f.remaining != "" {
					w.Header().Set("X-RateLimit-Remaining", f.remaining)
					w.Header().Set("X-RateLimit-Reset", "1900000000")
				}
				w.WriteHeader(f.workflowStatus)
				return
			}
			parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/repos/"), "/contents/", 2)
			key := parts[0] + "@" + r.URL.Query().Get("ref") + ":" + parts[1]
			text, ok := b.Files[key]
			if !ok {
				w.WriteHeader(404)
				return
			}
			enc.Encode(map[string]string{"content": base64.StdEncoding.EncodeToString([]byte(text)), "encoding": "base64"})
		case strings.HasSuffix(r.URL.Path, "/logs"):
			http.SetCookie(w, &http.Cookie{Name: "api-session", Value: "not-forwarded"})
			w.Header().Set("Location", fmt.Sprintf("%s/signed?signature=do-not-cache-%d", f.log.URL, f.counts[r.URL.Path]))
			w.WriteHeader(302)
		case strings.HasSuffix(r.URL.Path, "/jobs"):
			want := fmt.Sprintf("/attempts/%d/jobs", b.RunData.RunAttempt)
			if f.attempt > 0 {
				want = fmt.Sprintf("/attempts/%d/jobs", f.attempt)
			}
			if !strings.HasSuffix(r.URL.Path, want) {
				t.Errorf("wrong attempt endpoint %s", r.URL.Path)
			}
			if f.pages {
				jobs := []github.Job{}
				page := r.URL.Query().Get("page")
				n := 100
				if page == "2" {
					n = 1
				}
				for i := 0; i < n; i++ {
					jobs = append(jobs, github.Job{ID: int64(i + 1)})
					if page == "2" {
						jobs[i].ID = 101
					}
				}
				enc.Encode(map[string]any{"jobs": jobs})
			} else {
				enc.Encode(map[string]any{"jobs": b.JobData})
			}
		case strings.Contains(r.URL.Path, "/actions/workflows/"):
			enc.Encode(map[string]string{"path": b.RunData.Path})
		default:
			run := b.RunData
			if f.attempt > 0 {
				run.RunAttempt = f.attempt
			}
			enc.Encode(run)
		}
	}))
	t.Cleanup(f.api.Close)
	t.Cleanup(f.log.Close)
	return f
}
func (f *fixture) source(root string, refresh bool) *Source {
	s, e := New(Options{Root: root, Refresh: refresh})
	if e != nil {
		f.t.Fatal(e)
	}
	s.HTTP.Base = f.api.URL
	s.HTTP.Download.Transport = f.log.Client().Transport
	return s
}
func (f *fixture) resolve(s *Source) error {
	l, e := resolver.Resolve(context.Background(), s, f.bundle.URL, resolver.Options{})
	if e == nil && l.Commit == "" {
		f.t.Fatal("empty resolver output")
	}
	return s.Finish(e)
}
func TestOnlineBundleParityRawCacheAndRefresh(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "sent-only-to-api")
	t.Setenv("GH_TOKEN", "fallback-must-not-win")
	f := newFixture(t)
	f.auth = "Bearer sent-only-to-api"
	f.gzip = true
	root := t.TempDir()
	s := f.source(root, false)
	defer s.Close()
	onlineLock, e := resolver.Resolve(context.Background(), s, f.bundle.URL, resolver.Options{})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(nil); e != nil {
		t.Fatal(e)
	}
	offline, e := resolver.Resolve(context.Background(), f.bundle, f.bundle.URL, resolver.Options{})
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(onlineLock, offline) {
		t.Fatal("canonical source resolution differs")
	}
	var m Manifest
	raw, e := os.ReadFile(filepath.Join(s.CachePath, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &m)
	log, e := os.ReadFile(filepath.Join(s.CachePath, "job.raw.log"))
	if e != nil {
		t.Fatal(e)
	}
	if digest(log) != m.JobLogSHA || int64(len(log)) != m.JobLogSize || bytes.HasPrefix(log, []byte{0x1f, 0x8b}) {
		t.Fatal("raw hash/HTTP gzip handling")
	}
	downloads := f.counts["download"]
	hit := f.source(root, false)
	defer hit.Close()
	if e = f.resolve(hit); e != nil {
		t.Fatal(e)
	}
	if hit.CacheStatus != "HIT" || f.counts["download"] != downloads {
		t.Fatal("valid cache not used")
	}
	beforeRefresh := map[string]int{}
	for k, v := range f.counts {
		beforeRefresh[k] = v
	}
	fresh := f.source(root, true)
	defer fresh.Close()
	if e = f.resolve(fresh); e != nil {
		t.Fatal(e)
	}
	for path, n := range beforeRefresh {
		if f.counts[path] != n+1 {
			t.Errorf("refresh missed acquisition endpoint %s", path)
		}
	}
	if fresh.CacheStatus != "REFRESH" || f.counts["download"] != downloads+1 {
		t.Fatal("refresh did not redownload")
	}
	filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e == nil && !d.IsDir() && d.Type()&os.ModeSymlink == 0 {
			b, _ := os.ReadFile(path)
			for _, bad := range []string{"sent-only-to-api", "signature=do-not-cache", "fallback-must-not-win"} {
				if bytes.Contains(b, []byte(bad)) {
					t.Error("cached secret/redirect", path)
				}
			}
		}
		return nil
	})
}
func TestPartialAndCorruptCacheAreMisses(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	for _, mutate := range []string{"delete", "corrupt"} {
		t.Run(mutate, func(t *testing.T) {
			f := newFixture(t)
			root := t.TempDir()
			s := f.source(root, false)
			defer s.Close()
			if e := f.resolve(s); e != nil {
				t.Fatal(e)
			}
			if mutate == "delete" {
				os.Remove(filepath.Join(s.CachePath, "jobs.json"))
			} else {
				os.WriteFile(filepath.Join(s.CachePath, "job.raw.log"), []byte("corrupt"), 0600)
			}
			retry := f.source(root, false)
			defer retry.Close()
			if e := f.resolve(retry); e != nil {
				t.Fatal(e)
			}
			if retry.CacheStatus == "HIT" || f.counts["download"] != 2 {
				t.Fatal("incomplete/corrupt cache accepted")
			}
		})
	}
}
func TestAttemptsPaginationAndMetadataWorkflowPath(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	f := newFixture(t)
	f.attempt = 2
	f.pages = true
	s := f.source(t.TempDir(), false)
	defer s.Close()
	target, _ := github.ParseURL(f.bundle.URL + "/attempts/2")
	r, e := s.Run(context.Background(), target)
	if e != nil || r.RunAttempt != 2 {
		t.Fatal(r, e)
	}
	jobs, e := s.Jobs(context.Background(), target, 2)
	if e != nil || len(jobs) != 101 {
		t.Fatal(len(jobs), e)
	}
	// Missing path is resolved solely from workflow metadata.
	f.pages = false
	f.bundle.RunData.WorkflowID = 42
	path := f.bundle.RunData.Path
	f.bundle.RunData.Path = ""
	f.api.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/workflows/42") {
			json.NewEncoder(w).Encode(map[string]string{"path": path})
			return
		}
		json.NewEncoder(w).Encode(f.bundle.RunData)
	})
	other := f.source(t.TempDir(), false)
	defer other.Close()
	target.Attempt = 0
	got, e := other.Run(context.Background(), target)
	if e != nil || got.Path != path {
		t.Fatal(got.Path, e)
	}
}
func TestWorkflowErrorClassification(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	for _, tc := range []struct {
		status           int
		remaining, cause string
	}{{404, "", "WORKFLOW_UNAVAILABLE"}, {403, "0", "GITHUB_RATE_LIMITED"}, {403, "5", "GITHUB_FORBIDDEN"}} {
		f := newFixture(t)
		f.workflowStatus = tc.status
		f.remaining = tc.remaining
		s := f.source(t.TempDir(), false)
		defer s.Close()
		e := f.resolve(s)
		if e == nil || !strings.Contains(e.Error(), tc.cause) {
			t.Fatal(tc, e)
		}
	}
}

type zeros struct{ left int64 }

func (z *zeros) Read(p []byte) (int, error) {
	if z.left == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > z.left {
		n = int(z.left)
	}
	clear(p[:n])
	z.left -= int64(n)
	return n, nil
}
func TestStreamingDefaultLimitAndRawBytes(t *testing.T) {
	f, e := os.CreateTemp(t.TempDir(), "log")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	info, e := streamLog(&zeros{DefaultMaxLog + 1}, f, DefaultMaxLog, "")
	if e == nil || !strings.Contains(e.Error(), "EVIDENCE_TOO_LARGE") || info.Size <= DefaultMaxLog {
		t.Fatal(info, e)
	}
	raw := []byte("line\r\n\x1b[31mError\x1b[0m\n")
	f2, _ := os.CreateTemp(t.TempDir(), "raw")
	defer f2.Close()
	info, e = streamLog(bytes.NewReader(raw), f2, 1000, "")
	saved, _ := os.ReadFile(f2.Name())
	if e != nil || !bytes.Equal(saved, raw) || info.SHA256 != digest(raw) {
		t.Fatal(info, e)
	}
}
func TestLimitConfigurationAndTokenPriority(t *testing.T) {
	for _, bad := range []string{"", "0", "-1", "100MiB", "1.5", " 42"} {
		t.Setenv("RUNBACK_MAX_LOG_SIZE", bad)
		if _, e := NewHTTP(); e == nil {
			t.Fatal(bad)
		}
	}
	t.Setenv("RUNBACK_MAX_LOG_SIZE", "42")
	t.Setenv("GH_TOKEN", "fallback")
	t.Setenv("RUNBACK_GITHUB_TOKEN", "primary")
	h, e := NewHTTP()
	if e != nil || h.MaxLog != 42 || h.token != "primary" {
		t.Fatal(h, e)
	}
	t.Setenv("RUNBACK_GITHUB_TOKEN", "")
	h, _ = NewHTTP()
	if h.token != "fallback" {
		t.Fatal("fallback")
	}
	t.Setenv("GH_TOKEN", "")
	h, _ = NewHTTP()
	if h.Mode() != "anonymous" {
		t.Fatal("anonymous")
	}
}
func TestNetworkAndCredentialRedaction(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "redact-me")
	h, _ := NewHTTP()
	h.Base = "http://127.0.0.1:1"
	e := h.JSON(context.Background(), "/repos/a/b", "ACQUIRE", "RUN_METADATA_UNAVAILABLE", &struct{}{})
	if e == nil || !strings.Contains(e.Error(), "NETWORK") || strings.Contains(e.Error(), "redact-me") {
		t.Fatal(e)
	}
	f, _ := os.CreateTemp(t.TempDir(), "raw")
	defer f.Close()
	_, e = streamLog(strings.NewReader("redact-me"), f, 100, "redact-me")
	if e == nil || strings.Contains(e.Error(), "redact-me") {
		t.Fatal(e)
	}
}
func TestStaleCleanupSafety(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"runback-acquire-old", "runback-acquire-fresh", "user-data", "attempt-1"} {
		os.Mkdir(filepath.Join(root, name), 0700)
	}
	old := time.Now().Add(-25 * time.Hour)
	for _, name := range []string{"runback-acquire-old", "user-data", "attempt-1"} {
		os.Chtimes(filepath.Join(root, name), old, old)
	}
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(root, "runback-acquire-link"))
	CleanupTemps(root)
	if _, e := os.Stat(filepath.Join(root, "runback-acquire-old")); !os.IsNotExist(e) {
		t.Fatal("stale temp retained")
	}
	for _, name := range []string{"runback-acquire-fresh", "user-data", "attempt-1", "runback-acquire-link"} {
		if _, e := os.Lstat(filepath.Join(root, name)); e != nil {
			t.Fatal("unowned/fresh path removed", name)
		}
	}
}
