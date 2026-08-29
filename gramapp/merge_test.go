package main

import (
	"strings"
	"testing"
)

func TestMergeDuplicateAndConflict(t *testing.T) {
	facts := []rawFact{
		{Text: "시드 7억원 유치함", Location: "a.xlsx/part-1"},
		{Text: "시드 7억원 유치함", Location: "b.docx/part-1"},
		{Text: "매출 50억원 달성함", Location: "장표.pptx/slide-1"},
		{Text: "매출 70억원 달성함", Location: "엑셀.xlsx/part-1"},
		{Text: "NDA 체결함", Location: "메일.html/part-1"},
	}
	b := mergeFacts(facts, 4)
	if len(b.Shared) != 1 || !strings.Contains(itemLine(b.Shared[0]), "7억원") {
		t.Fatalf("shared %#v", b.Shared)
	}
	if len(b.Unique) != 1 || !strings.Contains(itemLine(b.Unique[0]), "NDA") {
		t.Fatalf("unique %#v", b.Unique)
	}
	if len(b.Conflict) != 1 || !strings.Contains(itemLine(b.Conflict[0]), "≠") {
		t.Fatalf("conflict %#v", b.Conflict)
	}
	if !strings.Contains(itemLine(b.Conflict[0]), "50억원") || !strings.Contains(itemLine(b.Conflict[0]), "70억원") {
		t.Fatalf("conflict sides %#v", b.Conflict)
	}
}

func TestMergeSingleFileIsShared(t *testing.T) {
	facts := []rawFact{
		{Text: "검토 필요함", Location: "a.pptx/slide-1"},
		{Text: "일정 확정함", Location: "a.pptx/slide-2"},
	}
	b := mergeFacts(facts, 1)
	if len(b.Unique) != 0 {
		t.Fatalf("unique on one file: %#v", b.Unique)
	}
	if len(b.Shared) != 2 {
		t.Fatalf("shared %#v", b.Shared)
	}
}

func TestFlattenAndPayload(t *testing.T) {
	flat := flattenChunkFacts([]map[string]any{
		{"heading_hint": "a.csv/part-1", "facts": []any{"시드 7억원 유치함"}},
		{"heading_hint": "b.md/part-1", "facts": []any{"시드 7억원 유치함", "NDA 체결함"}},
	})
	if len(flat) != 3 {
		t.Fatalf("flat %d", len(flat))
	}
	b := mergeFacts(flat, 2)
	p := formatMergePayload(b, true)
	if !strings.Contains(p, "[공유]") || !strings.Contains(p, "본문에 반복하지 말 것") {
		t.Fatalf("payload %s", p)
	}
	secs := appendMergeSections(nil, b, true)
	var heads []string
	for _, s := range secs {
		heads = append(heads, str(s["heading"]))
	}
	joined := strings.Join(heads, ",")
	if !strings.Contains(joined, "한쪽에만 있음") {
		t.Fatalf("sections %v", heads)
	}
}

func TestContainmentSameNumbers(t *testing.T) {
	b := mergeFacts([]rawFact{
		{Text: "시드 7억원", Location: "a.xlsx/part-1"},
		{Text: "시드 7억원 유치함", Location: "b.docx/part-1"},
	}, 2)
	if len(b.Shared) != 1 {
		t.Fatalf("want 1 shared, got %#v unique=%#v", b.Shared, b.Unique)
	}
	if !strings.Contains(b.Shared[0].Sides[0].Text, "유치함") {
		t.Fatalf("keep longer %#v", b.Shared)
	}
}

func TestCountInputFiles(t *testing.T) {
	if n := countInputFiles([]string{`C:\x\a.xlsx`, `C:\x\b.docx`}); n != 2 {
		t.Fatalf("%d", n)
	}
}
