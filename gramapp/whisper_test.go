package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidecarTxt(t *testing.T) {
	got := sidecarTxt(filepath.Join("x", "회의.mp4"))
	if filepath.Base(got) != "회의.txt" {
		t.Fatal(got)
	}
}

func TestUnzipTo(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "a.zip")
	f, err := os.Create(zp)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("Release/whisper-cli.exe")
	_, _ = w.Write([]byte("fake"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	dest := filepath.Join(dir, "out")
	if err := unzipTo(zp, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dest, "Release", "whisper-cli.exe"))
	if err != nil || string(b) != "fake" {
		t.Fatalf("%v %s", err, b)
	}
}

func TestIsWhisperBin(t *testing.T) {
	if !isWhisperBin("whisper-cli.exe") || !isWhisperBin("whisper-cpp-darwin-arm64") {
		t.Fatal("should detect cli names")
	}
	if isWhisperBin("ggml.dll") || isWhisperBin("ggml-small.bin") {
		t.Fatal("should skip libs/models")
	}
}

func TestNeedWhisperError(t *testing.T) {
	err := needWhisperError{Path: "/tmp/a.mp4"}
	if !strings.Contains(err.Error(), "a.mp4") {
		t.Fatal(err)
	}
}

func TestToolsDirCandidatesWindowsASCIIFirst(t *testing.T) {
	got := toolsDirCandidatesAt("windows", `C:\Users\ssh\AppData\Local`, "")
	if len(got) < 2 || !strings.Contains(got[0], `LocalLLM`) || strings.Contains(got[0], "로컬") {
		t.Fatalf("want ASCII first: %v", got)
	}
	if !strings.Contains(got[1], "로컬LLM") {
		t.Fatalf("want hangul fallback: %v", got)
	}
}

func TestMigrateToolsDirMovesHangul(t *testing.T) {
	root := t.TempDir()
	hangul := filepath.Join(root, "로컬LLM", "tools")
	ascii := filepath.Join(root, "LocalLLM", "tools")
	if err := os.MkdirAll(hangul, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hangul, "marker"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	migrateToolsDir([]string{ascii, hangul})
	b, err := os.ReadFile(filepath.Join(ascii, "marker"))
	if err != nil || string(b) != "ok" {
		t.Fatalf("moved? %v %s", err, b)
	}
	if _, err := os.Stat(hangul); err == nil {
		t.Fatal("old hangul tools still present")
	}
}

func TestHasNonASCII(t *testing.T) {
	if hasNonASCII(`C:\Users\ssh\AppData\Local\LocalLLM\tools`) {
		t.Fatal("ascii")
	}
	if !hasNonASCII(`C:\Users\ssh\AppData\Local\로컬LLM\tools`) {
		t.Fatal("hangul")
	}
}

func TestAsciiArgPathCopiesHangul(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "로컬LLM")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "ggml-small.bin")
	if err := os.WriteFile(src, []byte("model"), 0644); err != nil {
		t.Fatal(err)
	}
	got, cleanup, err := asciiArgPath(src)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if hasNonASCII(got) {
		t.Fatalf("still hangul: %s", got)
	}
	b, err := os.ReadFile(got)
	if err != nil || string(b) != "model" {
		t.Fatalf("%v %s", err, b)
	}
}
