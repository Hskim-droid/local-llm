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

func TestNeedWhisperError(t *testing.T) {
	err := needWhisperError{Path: "/tmp/a.mp4"}
	if !strings.Contains(err.Error(), "a.mp4") {
		t.Fatal(err)
	}
}
