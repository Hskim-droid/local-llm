package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	appVersion      = "0.8.7"
	reportIssuesURL = "https://github.com/Hskim-droid/local-llm/issues"
)

func appDir() string {
	return filepath.Dir(toolsDir())
}

func errorNotePath() string {
	return filepath.Join(appDir(), "오류.txt")
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
	b.WriteString("로컬LLM 오류 쪽지\n")
	b.WriteString("이 파일을 이슈에 붙이세요. 원문·워드·추출문은 붙이지 마세요.\n")
	b.WriteString("오류 신고는 " + reportIssuesURL + " 로 주세요.\n\n")
	b.WriteString("시각: " + when.Format(time.RFC3339) + "\n")
	b.WriteString("버전: " + appVersion + "\n")
	b.WriteString(fmt.Sprintf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	if p.ID != "" {
		b.WriteString("프로필: " + p.ID + " (" + p.Label + ")\n")
	}
	b.WriteString(fmt.Sprintf("RAM: %.0f GB (빈 자리 %.1f GB)\n", ramGB(), availGB()))
	if packID != "" {
		b.WriteString("팩: " + packID + "\n")
	} else {
		b.WriteString("팩: (없음)\n")
	}
	exts := inputExts(files)
	if len(exts) > 0 {
		b.WriteString("입력 확장자: " + strings.Join(exts, " ") + "\n")
	} else {
		b.WriteString("입력 확장자: (없음)\n")
	}
	b.WriteString("오류: " + scrubErr(msg, files) + "\n")
	return b.String()
}

func sayReportTo(notePath string) {
	say("오류 신고는 " + reportIssuesURL + " 로 주세요.")
	say("원문 파일은 보내지 마세요. 오류.txt 만 주세요.")
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
