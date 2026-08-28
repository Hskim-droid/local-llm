package main

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type segment struct {
	Text     string
	Source   string
	Location string
}

func extractPath(p string) ([]segment, error) {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".pptx":
		return extractPPTX(p)
	case ".ppt":
		return nil, fmt.Errorf("구형 .ppt는 안 됩니다. PPTX로 저장하세요")
	case ".pdf":
		return extractPDF(p)
	case ".txt", ".md":
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		t := strings.TrimSpace(string(b))
		if t == "" {
			return nil, nil
		}
		return []segment{{Text: t, Source: "txt", Location: "part-1"}}, nil
	case ".mp4", ".m4a", ".mov", ".wav", ".mp3", ".webm":
		return extractVideo(p)
	default:
		return nil, fmt.Errorf("지원하지 않는 형식: %s", ext)
	}
}

func extractVideo(p string) ([]segment, error) {
	side := strings.TrimSuffix(p, filepath.Ext(p)) + ".txt"
	b, err := os.ReadFile(side)
	if err != nil {
		return nil, fmt.Errorf("영상은 같은 이름 .txt(전사문)를 옆에 두면 됩니다: %s", filepath.Base(side))
	}
	t := strings.TrimSpace(string(b))
	if t == "" {
		return nil, nil
	}
	return []segment{{Text: t, Source: "video", Location: "transcript"}}, nil
}

func extractPPTX(p string) ([]segment, error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	re := regexp.MustCompile(`ppt/slides/slide(\d+)\.xml$`)
	type sl struct {
		n    int
		name string
	}
	var slides []sl
	for _, f := range zr.File {
		m := re.FindStringSubmatch(f.Name)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		slides = append(slides, sl{n: n, name: f.Name})
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].n < slides[j].n })
	var out []segment
	for _, s := range slides {
		var f *zip.File
		for _, zf := range zr.File {
			if zf.Name == s.name {
				f = zf
				break
			}
		}
		if f == nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		raw, _ := io.ReadAll(rc)
		rc.Close()
		text := xmlText(raw)
		if strings.TrimSpace(text) != "" {
			out = append(out, segment{Text: text, Source: "ppt", Location: fmt.Sprintf("slide-%d", s.n)})
		}
	}
	return out, nil
}

func xmlText(raw []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var parts []string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if ch, ok := tok.(xml.CharData); ok {
			t := strings.TrimSpace(string(ch))
			if t != "" {
				parts = append(parts, t)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func extractPDF(p string) ([]segment, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	text := pdfPlain(b)
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("PDF에서 글을 못 읽었습니다 (스캔본일 수 있음)")
	}
	return []segment{{Text: text, Source: "pdf", Location: "page-1"}}, nil
}

func pdfPlain(b []byte) string {
	reStream := regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	reStr := regexp.MustCompile(`\((?:\\.|[^\\)])*\)`)
	var parts []string
	for _, m := range reStream.FindAllSubmatch(b, -1) {
		raw := m[1]
		data := raw
		if zr, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
			inf, err := io.ReadAll(zr)
			zr.Close()
			if err == nil {
				data = inf
			}
		}
		for _, sm := range reStr.FindAll(data, -1) {
			s := string(sm)
			s = s[1 : len(s)-1]
			s = strings.ReplaceAll(s, `\n`, "\n")
			s = strings.ReplaceAll(s, `\r`, "")
			s = strings.ReplaceAll(s, `\(`, "(")
			s = strings.ReplaceAll(s, `\)`, ")")
			s = strings.TrimSpace(s)
			if len(s) >= 2 {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, " ")
}
