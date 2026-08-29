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

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
}

func TestExtractOfficeAndWeb(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "a.csv")
	_ = os.WriteFile(csv, []byte("항목,금액\n시드,7억원\n"), 0644)
	segs, err := extractPath(csv)
	if err != nil || !strings.Contains(segs[0].Text, "7억원") {
		t.Fatalf("csv %#v %v", segs, err)
	}

	htmlp := filepath.Join(dir, "a.html")
	_ = os.WriteFile(htmlp, []byte(`<html><script>x</script><p>매출 50억원</p></html>`), 0644)
	segs, err = extractPath(htmlp)
	if err != nil || !strings.Contains(segs[0].Text, "50억원") || strings.Contains(segs[0].Text, "script") {
		t.Fatalf("html %#v %v", segs, err)
	}

	xmlp := filepath.Join(dir, "a.xml")
	_ = os.WriteFile(xmlp, []byte(`<doc><t>NDA 체결</t></doc>`), 0644)
	segs, err = extractPath(xmlp)
	if err != nil || !strings.Contains(segs[0].Text, "NDA") {
		t.Fatalf("xml %#v %v", segs, err)
	}

	docx := filepath.Join(dir, "a.docx")
	writeZip(t, docx, map[string]string{"word/document.xml": `<w:t>핵심 50억원</w:t>`})
	segs, err = extractPath(docx)
	if err != nil || !strings.Contains(segs[0].Text, "50억원") {
		t.Fatalf("docx %#v %v", segs, err)
	}

	xlsx := filepath.Join(dir, "a.xlsx")
	writeZip(t, xlsx, map[string]string{"xl/sharedStrings.xml": `<t>7억원</t>`})
	segs, err = extractPath(xlsx)
	if err != nil || !strings.Contains(segs[0].Text, "7억원") {
		t.Fatalf("xlsx %#v %v", segs, err)
	}

	md := filepath.Join(dir, "a.md")
	_ = os.WriteFile(md, []byte("# 제목\n시드 7억원\n"), 0644)
	segs, err = extractPath(md)
	if err != nil || !strings.Contains(segs[0].Text, "7억원") {
		t.Fatalf("md %#v %v", segs, err)
	}

	hwpx := filepath.Join(dir, "a.hwpx")
	writeZip(t, hwpx, map[string]string{"Contents/section0.xml": `<hp:t>회의 안건</hp:t>`})
	segs, err = extractPath(hwpx)
	if err != nil || !strings.Contains(segs[0].Text, "회의") {
		t.Fatalf("hwpx %#v %v", segs, err)
	}

	_ = os.WriteFile(filepath.Join(dir, "old.hwp"), []byte("x"), 0644)
	_, err = extractPath(filepath.Join(dir, "old.hwp"))
	if err == nil || !strings.Contains(err.Error(), "HWPX") {
		t.Fatalf("hwp %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "old.xls"), []byte("x"), 0644)
	_, err = extractPath(filepath.Join(dir, "old.xls"))
	if err == nil || !strings.Contains(err.Error(), "XLSX") {
		t.Fatalf("xls %v", err)
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
