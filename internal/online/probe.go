package online

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// AccessStatus is the non-secret portion of GitHub API availability. The
// credential itself never leaves the online acquisition package.
type AccessStatus struct {
	Mode      string
	Limit     int
	Remaining int
	Reset     string
}

// ProbeAccess checks the GitHub API plane with the same credential priority as
// normal online acquisition. It does not acquire or execute repository data.
func ProbeAccess(ctx context.Context) (AccessStatus, error) {
	h, err := NewHTTP()
	if err != nil {
		return AccessStatus{}, err
	}
	return probeAccess(ctx, h)
}

func probeAccess(ctx context.Context, h *HTTP) (AccessStatus, error) {
	status := AccessStatus{Mode: h.Mode()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.Base+"/rate_limit", nil)
	if err != nil {
		return status, problem("ACQUIRE", "NETWORK", "invalid GitHub rate-limit request")
	}
	req.Header.Set("User-Agent", "RunBack/0.1")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}
	resp, err := h.API.Do(req)
	if err != nil {
		return status, problem("ACQUIRE", "NETWORK", "GitHub availability check failed; check connectivity")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return status, h.status(resp, "ACQUIRE", "RUN_METADATA_UNAVAILABLE")
	}
	var payload struct {
		Resources struct {
			Core struct {
				Limit     int   `json:"limit"`
				Remaining int   `json:"remaining"`
				Reset     int64 `json:"reset"`
			} `json:"core"`
		} `json:"resources"`
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(b) > 1<<20 || h.containsToken(b) || json.Unmarshal(b, &payload) != nil {
		return status, problem("ACQUIRE", "RUN_METADATA_UNAVAILABLE", "invalid GitHub rate-limit response")
	}
	status.Limit = payload.Resources.Core.Limit
	status.Remaining = payload.Resources.Core.Remaining
	if payload.Resources.Core.Reset > 0 {
		status.Reset = time.Unix(payload.Resources.Core.Reset, 0).UTC().Format(time.RFC3339)
	}
	return status, nil
}
