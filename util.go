package main

import (
	"strings"
	"time"
)

func split(s string, at int) (string, string) {
	if len(s) <= at {
		return s, ""
	}
	return s[:at], s[at:]
}

func formatDuration(d time.Duration) string {
	s := d.String()
	if strings.Contains(s, ".") {
		s = s[:strings.Index(s, ".")] + "s"
	}
	if strings.HasPrefix(s, "0h") {
		s = s[2:]
	}
	s = strings.ReplaceAll(s, "h0m", "h")
	s = strings.ReplaceAll(s, "m0s", "m")
	return s
}
