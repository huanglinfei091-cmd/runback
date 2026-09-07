package main

import "testing"

func TestVersionOutput(t *testing.T) {
	for input, want := range map[string]string{
		"dev":           "RunBack dev",
		"0.1.0-alpha":   "RunBack v0.1.0-alpha",
		"v0.1.0-alpha":  "RunBack v0.1.0-alpha",
		" 0.1.0-alpha ": "RunBack v0.1.0-alpha",
	} {
		if got := versionOutput(input); got != want {
			t.Errorf("versionOutput(%q)=%q want %q", input, got, want)
		}
	}
}
