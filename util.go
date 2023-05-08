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
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	return s
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "{nil}"
	}
	return t.Format("02.01.2006 15:04:05")
}

func ref[T any](t T) *T {
	return &t
}
