package runner

import "strings"

func ExecutorCause(log string) string {
	s := strings.ToLower(log)
	for _, v := range []string{"failed to fetch \"https://github.com/", "action cache failed to fetch", "failed to clone action"} {
		if strings.Contains(s, v) {
			return "ACTION_FETCH_FAILED"
		}
	}
	for _, v := range []string{"unsupported runs using", "unsupported node", "invalid workflow", "unknown action type"} {
		if strings.Contains(s, v) {
			return "ACT_INCOMPATIBILITY"
		}
	}
	for _, v := range []string{"tls handshake timeout", "connection refused", "connection reset by peer", "could not resolve", "i/o timeout", "network is unreachable", "failed to download", "unexpected eof"} {
		if strings.Contains(s, v) {
			return "NETWORK_DEPENDENCY"
		}
	}
	for _, v := range []string{"cannot connect to the docker daemon", "permission denied", "no space left on device"} {
		if strings.Contains(s, v) {
			return "ENVIRONMENT_MISMATCH"
		}
	}
	return "UNKNOWN"
}
