package main

import (
	"os"
	"path/filepath"
	"strings"
)

func writeExtractDump(outDir string, segs []segment, hv harvest) {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString("## ")
		b.WriteString(s.Location)
		b.WriteString("\n")
		b.WriteString(s.Text)
		b.WriteString("\n\n")
	}
	_ = os.WriteFile(filepath.Join(outDir, T("out.harvest.txt")), []byte(b.String()), 0644)
	dumpJSON(filepath.Join(outDir, T("out.json.harvest")), map[string]any{"numbers": hv.Raw})
}
