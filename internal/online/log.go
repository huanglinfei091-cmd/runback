package online

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type LogInfo struct {
	Size   int64
	SHA256 string
}

// streamLog keeps bounded chunks, including a small overlap for credential detection.
func streamLog(body io.Reader, file *os.File, limit int64, token string) (LogInfo, error) {
	var info LogInfo
	hash := sha256.New()
	buf := make([]byte, 64<<10)
	tail := ""
	for {
		n, e := body.Read(buf)
		if n > 0 {
			info.Size += int64(n)
			if info.Size > limit {
				return info, problem("JOB_LOG", "EVIDENCE_TOO_LARGE", fmt.Sprintf("observed size >= %d bytes; configured limit %d bytes", info.Size, limit))
			}
			if token != "" {
				piece := tail + string(buf[:n])
				if strings.Contains(piece, token) {
					return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "credential-bearing log rejected")
				}
				keep := len(token) - 1
				if keep > len(piece) {
					keep = len(piece)
				}
				tail = piece[len(piece)-keep:]
			}
			if _, err := file.Write(buf[:n]); err != nil {
				return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "cannot write raw log")
			}
			hash.Write(buf[:n])
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return info, problem("JOB_LOG", "NETWORK", "job log download interrupted")
		}
	}
	if info.Size == 0 {
		return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "empty job log")
	}
	info.SHA256 = fmt.Sprintf("%x", hash.Sum(nil))
	return info, nil
}
func (h *HTTP) Log(ctx context.Context, path, dir string) (LogInfo, error) {
	var info LogInfo
	resp, e := h.request(ctx, path, h.JobAPI)
	if e != nil {
		return info, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 302 {
		return info, h.status(resp, "JOB_LOG", "JOB_LOG_UNAVAILABLE")
	}
	location := resp.Header.Get("Location")
	u, e := url.Parse(location)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "invalid signed log location")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", location, nil)
	if e != nil {
		return info, problem("JOB_LOG", "NETWORK", "cannot request log")
	}
	// No API headers copied. Token env is not read here. Download client has no Jar.
	reply, e := h.Download.Do(req)
	if e != nil {
		return info, problem("JOB_LOG", "NETWORK", "signed log download failed")
	}
	defer reply.Body.Close()
	if reply.StatusCode != http.StatusOK {
		return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", fmt.Sprintf("job log could not be retrieved (HTTP %d)", reply.StatusCode))
	}
	f, e := os.CreateTemp(dir, ".raw-log-")
	if e != nil {
		return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "cannot create raw log tempfile")
	}
	name := f.Name()
	defer os.Remove(name)
	info, e = streamLog(reply.Body, f, h.MaxLog, h.token)
	if e != nil {
		f.Close()
		return info, e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "cannot sync raw log")
	}
	if e = f.Close(); e != nil {
		return info, e
	}
	if e = os.Rename(name, filepath.Join(dir, "job.raw.log")); e != nil {
		return info, problem("JOB_LOG", "JOB_LOG_UNAVAILABLE", "cannot publish raw log")
	}
	return info, nil
}
