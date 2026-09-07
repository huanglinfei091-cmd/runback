package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseURL(t *testing.T) {
	for _, s := range []string{"https://github.com/o/r/actions/runs/123", "https://github.com/o/r/actions/runs/123/attempts/2?check_suite_focus=true"} {
		v, e := ParseURL(s)
		if e != nil || v.RunID != 123 {
			t.Fatalf("%v %v", v, e)
		}
	}
	for _, s := range []string{"http://github.com/o/r/actions/runs/1", "https://github.com.evil/o/r/actions/runs/1", "https://x@github.com/o/r/actions/runs/1", "https://github.com/o/r/actions/runs/0", "https://github.com/o/r/actions/runs/999999999999999999999", "https://github.com/../r/actions/runs/1", "https://github.com/./r/actions/runs/1", "https://github.com/o/r/actions/runs/1/attempts/0", "https://github.com/o/r/actions/runs/1/attempts/-1"} {
		if _, e := ParseURL(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestAttemptPagination(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/repos/o/r/actions/runs/9/attempts/2/jobs" {
			t.Error(r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Error("missing auth")
		}
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `{"jobs":[`)
			for i := 0; i < 100; i++ {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprint(w, `{"id":1}`)
			}
			fmt.Fprint(w, "]}")
		} else {
			fmt.Fprint(w, `{"jobs":[{"id":2}]}`)
		}
	}))
	defer s.Close()
	c := &Client{BaseURL: s.URL, Token: "test", HTTP: s.Client()}
	j, e := c.Jobs(context.Background(), Target{Owner: "o", Repo: "r", RunID: 9}, 2)
	if e != nil || len(j) != 101 || calls != 2 {
		t.Fatalf("%d %d %v", len(j), calls, e)
	}
}
func TestAPIError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer s.Close()
	c := &Client{BaseURL: s.URL, HTTP: s.Client()}
	if _, e := c.Get(context.Background(), "/rate-limit"); e == nil {
		t.Fatal("expected permission error")
	}
}
