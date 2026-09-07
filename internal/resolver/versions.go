package resolver

import (
	"regexp"
	"strings"
)

// ObservedVersions returns unambiguous exact runtime versions reported by setup actions.
func ObservedVersions(log string) map[string]string {
	patterns := map[string]*regexp.Regexp{
		"uv-version":     regexp.MustCompile(`Successfully installed uv version (\d+\.\d+\.\d+)\b`),
		"node-version":   regexp.MustCompile(`(?m)\bnode: v(\d+\.\d+\.\d+)\b`),
		"python-version": regexp.MustCompile(`Successfully set up CPython \((\d+\.\d+\.\d+)\)`),
		"go-version":     regexp.MustCompile(`\bgo version go(\d+\.\d+(?:\.\d+)?) linux/amd64`),
	}
	out := map[string]string{}
	for key, re := range patterns {
		matches := re.FindAllStringSubmatch(log, -1)
		unique := map[string]bool{}
		for _, m := range matches {
			unique[strings.TrimSpace(m[1])] = true
		}
		if len(unique) == 1 {
			for v := range unique {
				out[key] = v
			}
		}
	}
	return out
}
