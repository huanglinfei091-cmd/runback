package online

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeAccessUsesOnlineCredentialWithoutReturningIt(t *testing.T) {
	const secret = "doctor-online-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rate_limit" || r.Header.Get("Authorization") != "Bearer "+secret {
			t.Fatalf("unexpected request path=%q auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		fmt.Fprint(w, `{"resources":{"core":{"limit":5000,"remaining":4321,"reset":1900000000}}}`)
	}))
	defer server.Close()
	h, err := NewHTTP()
	if err != nil {
		t.Fatal(err)
	}
	h.Base = server.URL
	h.token = secret
	status, err := probeAccess(context.Background(), h)
	if err != nil || status.Mode != "token" || status.Limit != 5000 || status.Remaining != 4321 || status.Reset == "" {
		t.Fatal(status, err)
	}
	if strings.Contains(fmt.Sprint(status), secret) {
		t.Fatal("credential escaped online package")
	}
}

func TestProbeAccessAnonymousAndRateClassification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Fatal("anonymous probe sent authorization")
		}
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1900000000")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	h, _ := NewHTTP()
	h.Base = server.URL
	h.token = ""
	status, err := probeAccess(context.Background(), h)
	if status.Mode != "anonymous" || err == nil || !strings.Contains(err.Error(), "GITHUB_RATE_LIMITED") || strings.Contains(err.Error(), "Bearer") {
		t.Fatal(status, err)
	}
}
