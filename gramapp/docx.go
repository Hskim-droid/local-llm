package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func writeDocx(path string, title string, header [][]string, sections []map[string]any) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)
	p(&b, title, 32, true, "center")
	for _, row := range header {
		if len(row) >= 2 && strings.TrimSpace(row[1]) != "" && row[1] != "(원문에 없음)" {
			p(&b, row[0]+"  "+row[1], 21, false, "left")
		}
		if len(row) == 4 && strings.TrimSpace(row[3]) != "" && row[3] != "(원문에 없음)" {
			p(&b, row[2]+"  "+row[3], 21, false, "left")
		}
	}
	p(&b, "회의 내용", 24, true, "left")
	for _, sec := range sections {
		h, _ := sec["heading"].(string)
		if h != "" {
			p(&b, h, 24, true, "left")
		}
		blocks, _ := sec["blocks"].([]any)
		if blocks == nil {
			if bb, ok := sec["blocks"].([]map[string]any); ok {
				for _, bl := range bb {
					writeBlock(&b, bl)
				}
			}
		} else {
			for _, raw := range blocks {
				if bl, ok := raw.(map[string]any); ok {
					writeBlock(&b, bl)
				}
			}
		}
	}
	p(&b, "끝.", 21, false, "right")
	b.WriteString(`</w:body></w:document>`)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`,
		"word/document.xml": b.String(),
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return err
		}
	}
	return zw.Close()
}

func writeBlock(b *strings.Builder, bl map[string]any) {
	if bullets, ok := asStringSlice(bl["bullets"]); ok {
		for _, x := range bullets {
			p(b, "·  "+x, 21, false, "left")
		}
	}
	if ordered, ok := asStringSlice(bl["ordered"]); ok {
		for i, x := range ordered {
			p(b, fmt.Sprintf("%d.  %s", i+1, x), 21, false, "left")
		}
	}
	if para, ok := bl["para"].(string); ok && para != "" {
		p(b, para, 21, false, "left")
	}
}

func asStringSlice(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return t, true
	case []any:
		var out []string
		for _, x := range t {
			switch y := x.(type) {
			case string:
				out = append(out, y)
			case map[string]any:
				if s, ok := y["li"].(string); ok {
					out = append(out, s)
				}
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func p(b *strings.Builder, text string, halfPoints int, bold bool, align string) {
	text = escapeXML(text)
	al := "left"
	if align == "center" {
		al = "center"
	}
	if align == "right" {
		al = "right"
	}
	b.WriteString(`<w:p><w:pPr><w:jc w:val="` + al + `"/><w:spacing w:after="80"/></w:pPr><w:r><w:rPr>`)
	if bold {
		b.WriteString(`<w:b/>`)
	}
	b.WriteString(fmt.Sprintf(`<w:sz w:val="%d"/><w:szCs w:val="%d"/><w:rFonts w:ascii="Malgun Gothic" w:eastAsia="Malgun Gothic"/>`, halfPoints, halfPoints))
	b.WriteString(`</w:rPr><w:t xml:space="preserve">`)
	b.WriteString(text)
	b.WriteString(`</w:t></w:r></w:p>`)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func dumpJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0644)
}
