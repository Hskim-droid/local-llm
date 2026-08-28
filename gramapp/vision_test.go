package main

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if !imageOK(b) {
		t.Fatalf("sample jpeg not ok, len=%d", len(b))
	}
	return b
}

func TestImageOKRejectsTiny(t *testing.T) {
	if imageOK([]byte{0xFF, 0xD8, 0xFF, 0xD9}) {
		t.Fatal("tiny jpeg should fail")
	}
}

func TestFindJPEGsInPDFBytes(t *testing.T) {
	jpg := sampleJPEG(t)
	blob := append([]byte("%PDF-1.4 junk "), jpg...)
	blob = append(blob, []byte(" end")...)
	got := findJPEGs(blob)
	if len(got) != 1 {
		t.Fatalf("want 1 jpeg, got %d", len(got))
	}
	if len(got[0]) != len(jpg) {
		t.Fatalf("size %d vs %d", len(got[0]), len(jpg))
	}
}

func TestPPTXChartSlideCollectsImage(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "chart.pptx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	jpg := sampleJPEG(t)
	zw := zip.NewWriter(f)
	w, _ := zw.Create("ppt/slides/slide1.xml")
	_, _ = w.Write([]byte(`<p:sld><a:t>차트</a:t></p:sld>`))
	w, _ = zw.Create("ppt/slides/_rels/slide1.xml.rels")
	_, _ = w.Write([]byte(`<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/chart.jpg"/>
</Relationships>`))
	w, _ = zw.Create("ppt/media/chart.jpg")
	_, _ = w.Write(jpg)
	w, _ = zw.Create("ppt/slides/slide2.xml")
	_, _ = w.Write([]byte(`<p:sld><a:t>이 슬라이드는 글자가 충분히 많아서 그림 모델을 부르지 않아야 합니다. 50억원.</a:t></p:sld>`))
	w, _ = zw.Create("ppt/slides/_rels/slide2.xml.rels")
	_, _ = w.Write([]byte(`<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/logo.jpg"/>
</Relationships>`))
	w, _ = zw.Create("ppt/media/logo.jpg")
	_, _ = w.Write(jpg)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	imgs := pptxImages(p)
	if len(imgs) != 1 {
		t.Fatalf("want only thin slide image, got %d", len(imgs))
	}
	if imgs[0].Location != "slide-1" {
		t.Fatalf("loc %s", imgs[0].Location)
	}
}

func TestPDFImagesSkipWhenTextRich(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.pdf")
	jpg := sampleJPEG(t)
	var body strings.Builder
	body.WriteString("%PDF-1.4\n")
	long := strings.Repeat("매출 50억원 보고. ", 40)
	body.WriteString("stream\n(" + long + ")\nendstream\n")
	body.Write(jpg)
	if err := os.WriteFile(p, []byte(body.String()), 0644); err != nil {
		t.Fatal(err)
	}
	if imgs := pdfImages(p); len(imgs) != 0 {
		t.Fatalf("rich text should skip vision, got %d", len(imgs))
	}
}

func TestPDFImagesOnScan(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "scan.pdf")
	jpg := sampleJPEG(t)
	blob := append([]byte("%PDF-1.4\nstream\n(Hi)\nendstream\n"), jpg...)
	if err := os.WriteFile(p, blob, 0644); err != nil {
		t.Fatal(err)
	}
	imgs := pdfImages(p)
	if len(imgs) != 1 {
		t.Fatalf("scan should yield jpeg, got %d", len(imgs))
	}
}
