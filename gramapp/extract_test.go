package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPPTX(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.pptx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("ppt/slides/slide1.xml")
	_, _ = w.Write([]byte(`<p:sld><a:t>전시회 핵심 50억원</a:t></p:sld>`))
	_ = zw.Close()
	_ = f.Close()
	segs, err := extractPPTX(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 || !strings.Contains(segs[0].Text, "50억원") {
		t.Fatalf("got %#v", segs)
	}
}

func TestExtractVideoSidecar(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "회의.mp4")
	if err := os.WriteFile(vid, []byte("not-a-video"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "회의.txt"), []byte("[00:12] 시드 7억원 논의함.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	segs, err := extractVideo(vid)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 || segs[0].Source != "video" || !strings.Contains(segs[0].Text, "7억원") {
		t.Fatalf("%#v", segs)
	}
}

func TestExtractVideoNeedsWhisper(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "회의.mp4")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := extractVideo(vid)
	if _, ok := err.(needWhisperError); !ok {
		t.Fatalf("want needWhisperError, got %v", err)
	}
}

func TestWriteDocx(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "보고서.docx")
	err := writeDocx(out, "제목", "회의 내용", [][]string{{"일 시", "2026.1.1", "회의시간", "10:00 ~ 11:00"}}, []map[string]any{
		{"heading": "0. Executive Summary", "blocks": []map[string]any{{"bullets": []string{"핵심임"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(out)
	if err != nil || st.Size() < 100 {
		t.Fatal("docx missing")
	}
}
