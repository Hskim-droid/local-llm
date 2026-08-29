package main

import (
	"archive/zip"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func extractPlain(p, source string) ([]segment, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	t := strings.TrimSpace(string(b))
	if t == "" {
		return nil, nil
	}
	return []segment{{Text: t, Source: source, Location: "part-1"}}, nil
}

func extractHTML(p string) ([]segment, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	t := htmlText(string(b))
	if t == "" {
		return nil, nil
	}
	return []segment{{Text: t, Source: "html", Location: "part-1"}}, nil
}

func extractXMLFile(p string) ([]segment, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	t := strings.TrimSpace(xmlText(b))
	if t == "" {
		return nil, nil
	}
	return []segment{{Text: t, Source: "xml", Location: filepath.Base(p)}}, nil
}

func htmlText(s string) string {
	reBlock := regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)>`)
	s = reBlock.ReplaceAllString(s, " ")
	reTag := regexp.MustCompile(`(?s)<[^>]+>`)
	s = reTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = regexp.MustCompile(`[ \t\r\n]+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func extractDOCX(p string) ([]segment, error) {
	return extractZipXML(p, "docx", func(name string) bool {
		n := strings.ToLower(name)
		if !strings.HasSuffix(n, ".xml") {
			return false
		}
		return strings.HasPrefix(n, "word/document") || strings.HasPrefix(n, "word/header") || strings.HasPrefix(n, "word/footer")
	})
}

func extractXLSX(p string) ([]segment, error) {
	return extractZipXML(p, "xlsx", func(name string) bool {
		n := strings.ToLower(name)
		if n == "xl/sharedstrings.xml" {
			return true
		}
		return strings.HasPrefix(n, "xl/worksheets/") && strings.HasSuffix(n, ".xml")
	})
}

func extractHWPX(p string) ([]segment, error) {
	return extractZipXML(p, "hwpx", func(name string) bool {
		n := strings.ToLower(name)
		if strings.HasSuffix(n, ".xml") && (strings.Contains(n, "contents/") || strings.Contains(n, "section")) {
			return true
		}
		return strings.HasSuffix(n, "prvtext.txt")
	})
}

func extractZipXML(p, source string, want func(name string) bool) ([]segment, error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var parts []string
	for _, f := range zr.File {
		if !want(f.Name) {
			continue
		}
		raw, err := readZipFile(f)
		if err != nil {
			continue
		}
		var t string
		if strings.HasSuffix(strings.ToLower(f.Name), ".txt") {
			t = strings.TrimSpace(string(raw))
		} else {
			t = strings.TrimSpace(xmlText(raw))
		}
		if t != "" {
			parts = append(parts, t)
		}
	}
	blob := strings.TrimSpace(strings.Join(parts, "\n"))
	if blob == "" {
		return nil, fmt.Errorf("%s에서 글을 못 읽었습니다", strings.ToUpper(source))
	}
	return []segment{{Text: blob, Source: source, Location: "part-1"}}, nil
}
