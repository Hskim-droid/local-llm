package main

import (
	"strings"
	"testing"
	"time"
)

func TestScrubErrHidesPaths(t *testing.T) {
	files := []string{"/Users/hskim/회사비밀/장표.pptx"}
	got := scrubErr("건너뜀 /Users/hskim/회사비밀/장표.pptx — 글을 못 읽었습니다", files)
	if strings.Contains(got, "회사비밀") || strings.Contains(got, "장표.pptx") || strings.Contains(got, "/Users/hskim") {
		t.Fatalf("leaked: %s", got)
	}
	if !strings.Contains(got, ".pptx") {
		t.Fatalf("want ext: %s", got)
	}
}

func TestFormatErrorNote(t *testing.T) {
	setUILang("en")
	p := profile{ID: "gram16", Label: "그램 16GB"}
	s := formatErrorNote(time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC), p, "보고서", "꺼낼 글이 없습니다", []string{`C:\work\a.xlsx`, `C:\work\b.hwpx`})
	if !strings.Contains(s, reportIssuesURL) || !strings.Contains(s, appVersion) {
		t.Fatalf("%s", s)
	}
	if !strings.Contains(s, ".xlsx") || !strings.Contains(s, ".hwpx") {
		t.Fatalf("exts %s", s)
	}
	if strings.Contains(s, `C:\work`) || strings.Contains(s, "a.xlsx") {
		t.Fatalf("path leaked %s", s)
	}
	if !strings.Contains(s, "보고서") {
		t.Fatalf("%s", s)
	}
}

func TestInputExts(t *testing.T) {
	got := inputExts([]string{"a.XLSX", "b.xlsx", "c.hwpx"})
	if strings.Join(got, " ") != ".xlsx .hwpx" {
		t.Fatalf("%v", got)
	}
}
