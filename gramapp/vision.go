package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const sysVL = "너는 스캔·차트 읽기다. 보이는 글과 숫자만 한국어 개조식으로. 창작 금지. 없는 수치를 만들지 말 것. JSON만. {\"facts\":[\"명사형\"],\"numbers\":[\"원문 그대로\"]}"

type imgPart struct {
	Location string
	Ext      string
	Bytes    []byte
}

func extractVisuals(p string) []imgPart {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".pptx":
		return pptxImages(p)
	case ".pdf":
		return pdfImages(p)
	default:
		return nil
	}
}

func pptxImages(p string) []imgPart {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil
	}
	defer zr.Close()
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	reSlide := regexp.MustCompile(`ppt/slides/slide(\d+)\.xml$`)
	type sl struct{ n int }
	var slides []sl
	for name := range files {
		m := reSlide.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		slides = append(slides, sl{n: n})
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].n < slides[j].n })
	var out []imgPart
	for _, s := range slides {
		slideName := fmt.Sprintf("ppt/slides/slide%d.xml", s.n)
		text := ""
		if f := files[slideName]; f != nil {
			if raw, err := readZipFile(f); err == nil {
				text = xmlText(raw)
			}
		}
		if utf8.RuneCountInString(strings.TrimSpace(text)) >= 40 {
			continue
		}
		relName := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", s.n)
		f := files[relName]
		if f == nil {
			continue
		}
		raw, err := readZipFile(f)
		if err != nil {
			continue
		}
		for _, tgt := range relTargets(raw) {
			name := path.Clean(path.Join("ppt/slides", tgt))
			zf := files[name]
			if zf == nil {
				continue
			}
			ext := strings.ToLower(path.Ext(name))
			if ext != ".jpeg" && ext != ".jpg" && ext != ".png" {
				continue
			}
			b, err := readZipFile(zf)
			if err != nil || !imageOK(b) {
				continue
			}
			out = append(out, imgPart{Location: fmt.Sprintf("slide-%d", s.n), Ext: ext, Bytes: b})
			if len(out) >= 8 {
				return out
			}
		}
	}
	return out
}

func relTargets(raw []byte) []string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var out []string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		var tgt, typ string
		for _, a := range se.Attr {
			switch a.Name.Local {
			case "Target":
				tgt = a.Value
			case "Type":
				typ = a.Value
			}
		}
		if tgt == "" {
			continue
		}
		if strings.Contains(typ, "/image") || strings.Contains(strings.ToLower(tgt), "/media/") {
			out = append(out, tgt)
		}
	}
	return out
}

func pdfImages(p string) []imgPart {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if utf8.RuneCountInString(strings.TrimSpace(pdfPlain(b))) >= 200 {
		return nil
	}
	var out []imgPart
	for i, jpg := range findJPEGs(b) {
		if !imageOK(jpg) {
			continue
		}
		out = append(out, imgPart{Location: fmt.Sprintf("scan-%d", i+1), Ext: ".jpg", Bytes: jpg})
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func findJPEGs(b []byte) [][]byte {
	var out [][]byte
	for i := 0; i < len(b)-2; i++ {
		if b[i] != 0xFF || b[i+1] != 0xD8 || b[i+2] != 0xFF {
			continue
		}
		for j := i + 3; j < len(b)-1; j++ {
			if b[j] == 0xFF && b[j+1] == 0xD9 {
				out = append(out, b[i:j+2])
				i = j + 1
				break
			}
		}
	}
	return out
}

func imageOK(b []byte) bool {
	if len(b) < 8*1024 || len(b) > 6<<20 {
		return false
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return len(b) >= 20*1024
	}
	return cfg.Width >= 120 && cfg.Height >= 80
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func fillVision(segs []segment, imgs []imgPart) []segment {
	if len(imgs) == 0 {
		return segs
	}
	say(fmt.Sprintf("글이 얇은 차트·스캔 %d장. 보고서 모델을 내리고 그림 모델을 올립니다. 동시에 안 올립니다.", len(imgs)))
	llamaStop()
	if err := ensureVL(); err != nil {
		say("  그림 모델 없음: " + err.Error() + "  — 글만 있는 대로 진행합니다.")
		return segs
	}
	for i, im := range imgs {
		say(fmt.Sprintf("  그림 %d/%d (%s)", i+1, len(imgs), im.Location))
		obj, err := llamaChatVision(sysVL, "보이는 숫자와 글만 JSON.", im.Bytes, im.Ext)
		if err != nil {
			say("    건너뜀: " + err.Error())
			continue
		}
		var parts []string
		if facts, ok := asStringSlice(obj["facts"]); ok {
			parts = append(parts, facts...)
		}
		if nums, ok := asStringSlice(obj["numbers"]); ok {
			parts = append(parts, nums...)
		}
		t := strings.TrimSpace(strings.Join(parts, "\n"))
		if t == "" {
			continue
		}
		segs = append(segs, segment{Text: t, Source: "vision", Location: im.Location})
	}
	say("그림 모델을 내렸습니다. 이제 보고서 모델만 씁니다.")
	return segs
}

func ensureVL() error {
	if err := ensureLlamaBin(); err != nil {
		return err
	}
	say("  스캔·차트용 그림 모델입니다. 보고서 모델과 같이 올리지 않습니다.")
	if err := ensureGGUF(vlGGUF); err != nil {
		return err
	}
	if err := ensureGGUF(vlProj); err != nil {
		return err
	}
	return nil
}
