package online

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huanglinfei091-cmd/runback/internal/github"
	"github.com/huanglinfei091-cmd/runback/internal/resolver"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func anonymous(t *testing.T) {
	t.Helper()
	t.Setenv("GH_TOKEN", "")
	t.Setenv("RUNBACK_GITHUB_TOKEN", "")
}
func TestRunStateAndIdentityRejections(t *testing.T) {
	anonymous(t)
	for _, kind := range []string{"queued", "in_progress", "success", "private", "attempt", "repo", "run", "no-path"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t)
			target, _ := github.ParseURL(f.bundle.URL + "/attempts/1")
			switch kind {
			case "queued", "in_progress":
				f.bundle.RunData.Status = kind
			case "success":
				f.bundle.RunData.Conclusion = "success"
			case "private":
				f.bundle.RunData.Repository.Private = true
			case "attempt":
				f.attempt = 2
			case "repo":
				f.bundle.RunData.Repository.FullName = "different/repository"
			case "run":
				f.bundle.RunData.ID++
			case "no-path":
				f.bundle.RunData.Path = ""
				f.bundle.RunData.WorkflowID = 0
			}
			s := f.source(t.TempDir(), false)
			defer s.Close()
			if _, e := s.Run(context.Background(), target); e == nil || !strings.Contains(e.Error(), "EVIDENCE_UNAVAILABLE") {
				t.Fatal(e)
			}
			if f.counts["download"] != 0 {
				t.Fatal("invalid run executed download")
			}
		})
	}
	f := newFixture(t)
	f.attempt = 3
	s := f.source(t.TempDir(), false)
	defer s.Close()
	target, _ := github.ParseURL(f.bundle.URL)
	r, e := s.Run(context.Background(), target)
	if e != nil || r.RunAttempt != 3 {
		t.Fatal(r, e)
	}
	if !strings.HasSuffix(s.CachePath, "attempt-3") {
		t.Fatal(s.CachePath)
	}
}
func TestHTTPStreamFailuresRemoveIncompleteEvidence(t *testing.T) {
	anonymous(t)
	for _, kind := range []string{"too-large", "interrupted", "empty", "non-redirect", "signed-error"} {
		t.Run(kind, func(t *testing.T) {
			log := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
					t.Error("credential leak")
				}
				switch kind {
				case "interrupted":
					w.Header().Set("Content-Length", "100")
					fmt.Fprint(w, "cut")
				case "too-large":
					fmt.Fprint(w, strings.Repeat("x", 65))
				case "signed-error":
					w.WriteHeader(404)
				default:
				}
			}))
			defer log.Close()
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if kind == "non-redirect" {
					fmt.Fprint(w, "plain body")
					return
				}
				w.Header().Set("Location", log.URL+"/signed")
				w.WriteHeader(302)
			}))
			defer api.Close()
			h, _ := NewHTTP()
			h.Base = api.URL
			h.MaxLog = 64
			h.Download.Transport = log.Client().Transport
			root := t.TempDir()
			_, e := h.Log(context.Background(), "/repos/a/b/actions/jobs/1/logs", root)
			if e == nil {
				t.Fatal("incomplete log accepted")
			}
			if kind == "too-large" && (!strings.Contains(e.Error(), "EVIDENCE_TOO_LARGE") || !strings.Contains(e.Error(), "64 bytes")) {
				t.Fatal(e)
			}
			if kind == "interrupted" && !strings.Contains(e.Error(), "NETWORK") {
				t.Fatal(e)
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 0 {
				t.Fatal("partial log retained", entries)
			}
		})
	}
}
func TestRefreshFailureKeepsPublishedCacheAndRemovesTemp(t *testing.T) {
	anonymous(t)
	f := newFixture(t)
	root := t.TempDir()
	initial := f.source(root, false)
	defer initial.Close()
	if e := f.resolve(initial); e != nil {
		t.Fatal(e)
	}
	old, _ := os.Readlink(initial.CachePath)
	fresh := f.source(root, true)
	_, e := resolver.Resolve(context.Background(), fresh, f.bundle.URL, resolver.Options{})
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(fresh.temp, "job.raw.log"), []byte("tampered"), 0600)
	if e = fresh.Finish(nil); e == nil {
		t.Fatal("tampered generation published")
	}
	temp := fresh.temp
	fresh.Close()
	after, _ := os.Readlink(initial.CachePath)
	if old != after {
		t.Fatal("failed refresh replaced cache")
	}
	if _, e := os.Stat(temp); !os.IsNotExist(e) {
		t.Fatal("failed acquisition temp retained")
	}
	hit := f.source(root, false)
	defer hit.Close()
	if e = f.resolve(hit); e != nil || hit.CacheStatus != "HIT" {
		t.Fatal(hit.CacheStatus, e)
	}
}
func TestJobsCannotMixAttemptsAndDuplicateIDs(t *testing.T) {
	anonymous(t)
	for _, kind := range []string{"attempt", "run", "duplicate"} {
		f := newFixture(t)
		switch kind {
		case "attempt":
			f.bundle.JobData[0].RunAttempt = 99
		case "run":
			f.bundle.JobData[0].RunID = 999
		case "duplicate":
			f.bundle.JobData = append(f.bundle.JobData, f.bundle.JobData[0])
		}
		s := f.source(t.TempDir(), false)
		defer s.Close()
		target, _ := github.ParseURL(f.bundle.URL)
		run, e := s.Run(context.Background(), target)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Jobs(context.Background(), target, run.RunAttempt); e == nil {
			t.Fatal(kind)
		}
	}
}
func TestAPIClientRedirectPolicyAndNoCookieJar(t *testing.T) {
	anonymous(t)
	h, _ := NewHTTP()
	if h.API == h.JobAPI || h.JobAPI == h.Download || h.API == h.Download {
		t.Fatal("shared client")
	}
	for _, c := range []*http.Client{h.API, h.JobAPI, h.Download} {
		if c.Jar != nil || c.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
			t.Fatal("redirect/cookie policy")
		}
	}
}

func TestSupplementalHistoricalWorkflowFailureStopsOnlineAcquisition(t *testing.T) {
	anonymous(t)
	f := newFixture(t)
	for key := range f.bundle.Files {
		parts := strings.SplitN(key, "@", 2)
		ref := strings.SplitN(parts[1], ":", 2)[0]
		if ref != f.bundle.RunData.HeadSHA {
			f.blockedRef = ref
			break
		}
	}
	if f.blockedRef == "" {
		t.Fatal("fixture lacks historical checkout workflow")
	}
	s := f.source(t.TempDir(), false)
	defer s.Close()
	e := f.resolve(s)
	if e == nil || !strings.Contains(e.Error(), "EVIDENCE_UNAVAILABLE") || !strings.Contains(e.Error(), "WORKFLOW_UNAVAILABLE") {
		t.Fatal(e)
	}
	if _, statErr := os.Stat(s.CachePath); !os.IsNotExist(statErr) {
		t.Fatal("incomplete online evidence was published")
	}
}
func TestRateLimitNeedsEvidenceAndNeverEchoesBody(t *testing.T) {
	t.Setenv("RUNBACK_GITHUB_TOKEN", "redact-this-secret")
	for _, tc := range []struct {
		status                  int
		remaining, retry, cause string
	}{
		{403, "", "", "GITHUB_FORBIDDEN"}, {403, "0", "", "GITHUB_RATE_LIMITED"}, {429, "", "2", "GITHUB_RATE_LIMITED"},
	} {
		api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-RateLimit-Remaining", tc.remaining)
			w.Header().Set("Retry-After", tc.retry)
			w.WriteHeader(tc.status)
			json.NewEncoder(w).Encode(map[string]string{"message": "redact-this-secret"})
		}))
		h, _ := NewHTTP()
		h.Base = api.URL
		e := h.JSON(context.Background(), "/repos/a/b", "WORKFLOW", "WORKFLOW_UNAVAILABLE", &struct{}{})
		api.Close()
		if e == nil || !strings.Contains(e.Error(), tc.cause) || strings.Contains(e.Error(), "redact-this-secret") {
			t.Fatal(e)
		}
	}
}
