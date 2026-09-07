// Package github retrieves immutable run evidence; it never executes repository code.
package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Target struct {
	Owner, Repo string
	RunID       int64
	Attempt     int
}

var runURL = regexp.MustCompile(`^/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)/actions/runs/([0-9]+)(?:/attempts/([0-9]+))?/?$`)

func ParseURL(raw string) (Target, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil {
		return Target{}, fmt.Errorf("expected https://github.com/owner/repo/actions/runs/ID")
	}
	m := runURL.FindStringSubmatch(u.Path)
	if m == nil || m[1] == ".." || m[2] == ".." || m[1] == "." || m[2] == "." {
		return Target{}, fmt.Errorf("invalid GitHub Actions run URL")
	}
	id, e := strconv.ParseInt(m[3], 10, 64)
	if e != nil || id <= 0 {
		return Target{}, fmt.Errorf("invalid run ID")
	}
	a := 0
	if m[4] != "" {
		a, e = strconv.Atoi(m[4])
		if e != nil || a < 1 {
			return Target{}, fmt.Errorf("invalid attempt")
		}
	}
	return Target{m[1], m[2], id, a}, nil
}
func (t Target) Repository() string { return t.Owner + "/" + t.Repo }
func (t Target) Base() string       { return "/repos/" + t.Repository() }

type Repository struct {
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}
type Ref struct {
	SHA  string     `json:"sha"`
	Ref  string     `json:"ref"`
	Repo Repository `json:"repo"`
}
type PR struct {
	Number int `json:"number"`
	Head   Ref `json:"head"`
	Base   Ref `json:"base"`
}
type Run struct {
	WorkflowID     int64      `json:"workflow_id"`
	ID             int64      `json:"id"`
	RunAttempt     int        `json:"run_attempt"`
	HeadSHA        string     `json:"head_sha"`
	HeadBranch     string     `json:"head_branch"`
	Event          string     `json:"event"`
	Path           string     `json:"path"`
	Name           string     `json:"name"`
	Conclusion     string     `json:"conclusion"`
	Status         string     `json:"status"`
	Repository     Repository `json:"repository"`
	HeadRepository Repository `json:"head_repository"`
	PullRequests   []PR       `json:"pull_requests"`
	Actor          struct {
		Login string `json:"login"`
	} `json:"actor"`
}
type Step struct {
	Name        string `json:"name"`
	Number      int    `json:"number"`
	Conclusion  string `json:"conclusion"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
}
type Job struct {
	RunID      int64    `json:"run_id"`
	RunAttempt int      `json:"run_attempt"`
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Conclusion string   `json:"conclusion"`
	Labels     []string `json:"labels"`
	Steps      []Step   `json:"steps"`
	RunnerName string   `json:"runner_name"`
}
type Client struct {
	BaseURL, Token string
	HTTP           *http.Client
}

func New() *Client {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b, e := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com").Output()
		if e == nil {
			token = strings.TrimSpace(string(b))
		}
	}
	return &Client{BaseURL: "https://api.github.com", Token: token, HTTP: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return fmt.Errorf("too many redirects")
		}
		if r.URL.Scheme != "https" {
			return fmt.Errorf("insecure redirect refused")
		}
		if r.URL.Host != via[0].URL.Host {
			r.Header.Del("Authorization")
		}
		return nil
	}}}
}
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(c.BaseURL, "/")+path, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "RunBack/0.1")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return nil, fmt.Errorf("GitHub request failed: %w", e)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub HTTP %d for %s (rate remaining %s; logs may require Actions read access or may have expired)", resp.StatusCode, strings.Split(path, "?")[0], resp.Header.Get("X-RateLimit-Remaining"))
	}
	const limit = 32 << 20
	b, e := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if len(b) > limit {
		return nil, fmt.Errorf("GitHub response exceeds 32 MiB")
	}
	return b, e
}
func (c *Client) JSON(ctx context.Context, path string, v any) error {
	b, e := c.Get(ctx, path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func (c *Client) Run(ctx context.Context, t Target) (Run, error) {
	p := fmt.Sprintf("%s/actions/runs/%d", t.Base(), t.RunID)
	if t.Attempt > 0 {
		p += fmt.Sprintf("/attempts/%d", t.Attempt)
	}
	var r Run
	e := c.JSON(ctx, p, &r)
	return r, e
}
func (c *Client) Jobs(ctx context.Context, t Target, attempt int) ([]Job, error) {
	var jobs []Job
	for page := 1; page <= 100; page++ {
		var p struct {
			Jobs []Job `json:"jobs"`
		}
		e := c.JSON(ctx, fmt.Sprintf("%s/actions/runs/%d/attempts/%d/jobs?per_page=100&page=%d", t.Base(), t.RunID, attempt, page), &p)
		if e != nil {
			return nil, e
		}
		jobs = append(jobs, p.Jobs...)
		if len(p.Jobs) < 100 {
			return jobs, nil
		}
	}
	return nil, fmt.Errorf("job pagination exceeded")
}
func (c *Client) File(ctx context.Context, repo, path, sha string) ([]byte, error) {
	var v struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	e := c.JSON(ctx, "/repos/"+repo+"/contents/"+path+"?ref="+url.QueryEscape(sha), &v)
	if e != nil {
		return nil, e
	}
	if v.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported content encoding")
	}
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(v.Content, "\n", ""))
}
