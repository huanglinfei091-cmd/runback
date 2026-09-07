// Package online acquires public GitHub evidence. It never executes repository code.
package online

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const DefaultMaxLog int64 = 104857600

type Error struct {
	Stage  string `json:"stage"`
	Cause  string `json:"cause"`
	Detail string `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	return "EVIDENCE_UNAVAILABLE\nStage: " + e.Stage + "\nCause: " + e.Cause + "\n" + e.Detail + "\nYou may retry using --bundle."
}
func problem(stage, cause, detail string) error { return &Error{stage, cause, detail} }

type HTTP struct {
	Base                  string
	token                 string
	API, JobAPI, Download *http.Client
	MaxLog                int64
}

func NewHTTP() (*HTTP, error) {
	limit := DefaultMaxLog
	if raw, ok := os.LookupEnv("RUNBACK_MAX_LOG_SIZE"); ok {
		n, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || n <= 0 {
			return nil, problem("CONFIG", "CONFIGURATION_ERROR", "RUNBACK_MAX_LOG_SIZE must be a positive integer in bytes")
		}
		limit = n
	}
	token := os.Getenv("RUNBACK_GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	// Three distinct clients; no cookie jars, no transport authentication middleware.
	noRedirect := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &HTTP{Base: "https://api.github.com", token: token, MaxLog: limit,
		API:      &http.Client{Timeout: 45 * time.Second, CheckRedirect: noRedirect},
		JobAPI:   &http.Client{Timeout: 45 * time.Second, CheckRedirect: noRedirect},
		Download: &http.Client{Timeout: 3 * time.Minute, CheckRedirect: noRedirect}}, nil
}
func (h *HTTP) Mode() string {
	if h.token != "" {
		return "token"
	}
	return "anonymous"
}
func (h *HTTP) containsToken(b []byte) bool {
	return h.token != "" && strings.Contains(string(b), h.token)
}
func (h *HTTP) status(resp *http.Response, stage, cause string) error {
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	limited := (resp.StatusCode == 403 || resp.StatusCode == 429) && (remaining == "0" || resp.Header.Get("Retry-After") != "")
	if limited {
		reset := resp.Header.Get("X-RateLimit-Reset")
		if n, e := strconv.ParseInt(reset, 10, 64); e == nil {
			reset = time.Unix(n, 0).UTC().Format(time.RFC3339)
		}
		// Header values are untrusted; never echo arbitrary tokens/URLs.
		if len(reset) > 40 || h.containsToken([]byte(reset)) {
			reset = "unavailable"
		}
		if _, e := strconv.Atoi(remaining); e != nil {
			remaining = "unavailable"
		}
		return problem("ACQUIRE", "GITHUB_RATE_LIMITED", fmt.Sprintf("remaining=%s reset=%s mode=%s. Set RUNBACK_GITHUB_TOKEN for a higher API limit.", remaining, reset, h.Mode()))
	}
	if resp.StatusCode == 403 {
		return problem(stage, "GITHUB_FORBIDDEN", "GitHub denied this request ("+h.Mode()+" mode)")
	}
	return problem(stage, cause, fmt.Sprintf("HTTP %d; requested evidence could not be retrieved", resp.StatusCode))
}
func (h *HTTP) request(ctx context.Context, path string, client *http.Client) (*http.Response, error) {
	if !strings.HasPrefix(path, "/repos/") {
		return nil, problem("ACQUIRE", "NETWORK", "invalid API path")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(h.Base, "/")+path, nil)
	if e != nil {
		return nil, problem("ACQUIRE", "NETWORK", "invalid API request")
	}
	req.Header.Set("User-Agent", "RunBack/0.1")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}
	resp, e := client.Do(req)
	if e != nil {
		return nil, problem("ACQUIRE", "NETWORK", "GitHub request failed; check connectivity")
	}
	return resp, nil
}
func (h *HTTP) JSON(ctx context.Context, path, stage, cause string, v any) error {
	resp, e := h.request(ctx, path, h.API)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return h.status(resp, stage, cause)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
	if e != nil {
		return problem(stage, "NETWORK", "metadata response interrupted")
	}
	if len(b) > 8<<20 {
		return problem(stage, cause, "metadata response exceeds limit")
	}
	if h.containsToken(b) {
		return problem(stage, cause, "credential-bearing response rejected")
	}
	if json.Unmarshal(b, v) != nil {
		return problem(stage, cause, "invalid metadata JSON")
	}
	return nil
}
func (h *HTTP) File(ctx context.Context, repo, path, sha string) ([]byte, error) {
	var v struct{ Content, Encoding string }
	e := h.JSON(ctx, "/repos/"+repo+"/contents/"+path+"?ref="+url.QueryEscape(sha), "WORKFLOW", "WORKFLOW_UNAVAILABLE", &v)
	if e != nil {
		return nil, e
	}
	if v.Encoding != "base64" {
		return nil, problem("WORKFLOW", "WORKFLOW_UNAVAILABLE", "unsupported workflow content encoding")
	}
	b, e := base64.StdEncoding.DecodeString(strings.ReplaceAll(v.Content, "\n", ""))
	if e != nil {
		return nil, problem("WORKFLOW", "WORKFLOW_UNAVAILABLE", "invalid workflow encoding")
	}
	if h.containsToken(b) {
		return nil, problem("WORKFLOW", "WORKFLOW_UNAVAILABLE", "credential-bearing content rejected")
	}
	return b, nil
}
