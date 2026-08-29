package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	appVersion      = "0.8.9"
	reportIssuesURL = "https://github.com/Hskim-droid/local-llm/issues"
)

func appDir() string {
	return filepath.Dir(toolsDir())
}

func errorNotePath() string {
	return filepath.Join(appDir(), T("out.error.file"))
}

func inputExts(files []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == "" || seen[ext] {
			continue
		}
		seen[ext] = true
		out = append(out, ext)
	}
	return out
}

func scrubErr(msg string, files []string) string {
	s := msg
	for _, f := range files {
		if f == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f))
		if ext == "" {
			ext = ".file"
		}
		s = strings.ReplaceAll(s, f, ext)
		base := filepath.Base(f)
		if base != "" && base != "." && base != string(filepath.Separator) {
			s = strings.ReplaceAll(s, base, ext)
		}
		dir := filepath.Dir(f)
		if dir != "" && dir != "." {
			s = strings.ReplaceAll(s, dir, "")
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		s = strings.ReplaceAll(s, home, "~")
	}
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		s = strings.ReplaceAll(s, d, "%LOCALAPPDATA%")
	}
	return strings.TrimSpace(s)
}

func formatErrorNote(when time.Time, p profile, packID, msg string, files []string) string {
	var b strings.Builder
	b.WriteString(T("note.title") + "\n")
	b.WriteString(T("note.body") + "\n")
	b.WriteString(T("note.send", reportIssuesURL) + "\n\n")
	b.WriteString(T("note.time", when.Format(time.RFC3339)) + "\n")
	b.WriteString(T("note.ver", appVersion) + "\n")
	b.WriteString(T("note.os", runtime.GOOS, runtime.GOARCH) + "\n")
	if p.ID != "" {
		b.WriteString(T("note.profile", p.ID, profileLabel(p)) + "\n")
	}
	b.WriteString(T("note.ram", ramGB(), availGB()) + "\n")
	if packID != "" {
		b.WriteString(T("note.pack", packID) + "\n")
	} else {
		b.WriteString(T("note.pack.none") + "\n")
	}
	exts := inputExts(files)
	if len(exts) > 0 {
		b.WriteString(T("note.ext", strings.Join(exts, " ")) + "\n")
	} else {
		b.WriteString(T("note.ext.none") + "\n")
	}
	b.WriteString(T("note.err", scrubErr(msg, files)) + "\n")
	return b.String()
}

func sayReportTo(notePath string) {
	say(T("report.send", reportIssuesURL))
	say(T("report.only"))
	if notePath != "" {
		say("  " + notePath)
	}
}

func writeErrorNote(p profile, packID, msg string, files []string) string {
	body := formatErrorNote(time.Now(), p, packID, msg, files)
	path := errorNotePath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return ""
	}
	return path
}

func reportFailure(p profile, packID string, files []string, err error) {
	if err == nil {
		return
	}
	path := writeErrorNote(p, packID, err.Error(), files)
	sayReportTo(path)
}
