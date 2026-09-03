package main

import "testing"

func TestSplitLongText(t *testing.T) {
	body := ""
	for i := 0; i < 50; i++ {
		body += "paragraph one is here.\n\n"
	}
	parts := splitLongText(body, 200)
	if len(parts) < 2 {
		t.Fatalf("parts %d", len(parts))
	}
	n := 0
	for _, p := range parts {
		n += len(p)
		if len(p) > 240 {
			t.Fatalf("chunk too big %d", len(p))
		}
	}
	if n < 400 {
		t.Fatalf("lost text %d", n)
	}
}

func TestSplitLongSegmentsLabels(t *testing.T) {
	segs := []segment{{Text: "", Location: "a.pdf/p1"}}
	big := ""
	for i := 0; i < 40; i++ {
		big += "line of source text for a pdf page.\n\n"
	}
	segs[0].Text = big
	got := splitLongSegments(segs, 200)
	if len(got) < 2 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Location != "a.pdf/p1#1" || got[1].Location != "a.pdf/p1#2" {
		t.Fatalf("%s %s", got[0].Location, got[1].Location)
	}
}

func TestAssembleSchemaRequired(t *testing.T) {
	s := assembleJSONSchema()
	req, _ := s["required"].([]string)
	if len(req) < 2 || req[0] != "title" {
		t.Fatalf("%v", s["required"])
	}
}
