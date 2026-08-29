package main

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedPacks(t *testing.T) {
	packs, err := loadPacksFromFS(embeddedPacks, "packs")
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) < 3 {
		t.Fatalf("want 3 packs, got %d", len(packs))
	}
	if packs[0].ID != "보고서" || packs[1].ID != "회의록" || packs[2].ID != "번역" {
		t.Fatalf("order: %#v %#v %#v", packs[0].ID, packs[1].ID, packs[2].ID)
	}
	for _, p := range packs {
		if p.ChunkSys == "" || p.AssembleSys == "" || p.OutSuffix == "" || p.OutName == "" {
			t.Fatalf("incomplete pack %+v", p)
		}
	}
	if p, ok := matchPack(packs, "2"); !ok || p.ID != "회의록" {
		t.Fatalf("match 2: %#v %v", p.ID, ok)
	}
	if p, ok := matchPack(packs, "번역"); !ok || p.OutSuffix != "_번역" {
		t.Fatalf("match 번역: %#v", p)
	}
}

func TestIsQuit(t *testing.T) {
	if !isQuit("끝") || !isQuit("QUIT") || isQuit("보고서") {
		t.Fatal("quit parse")
	}
}

func TestPackFileURL(t *testing.T) {
	u := packFileURL("보고서", "pack.json")
	if u == "" || !strings.Contains(u, "pack.json") {
		t.Fatal(u)
	}
}

func TestPackAccepts(t *testing.T) {
	packs, err := loadPacksFromFS(embeddedPacks, "packs")
	if err != nil {
		t.Fatal(err)
	}
	rep, _ := matchPack(packs, "보고서")
	min, _ := matchPack(packs, "회의록")
	for _, name := range []string{"a.pptx", "a.mp4", "a.xlsx", "a.csv", "a.md", "a.html", "a.xml", "a.hwpx", "a.docx"} {
		if !packAccepts(rep, name) || !packAccepts(min, name) {
			t.Fatalf("should take %s %+v", name, rep.Inputs)
		}
	}
	if packAccepts(rep, "a.hwp") || packAccepts(rep, "a.xls") || packAccepts(rep, "a.ppt") {
		t.Fatal("old binary office formats stay convert-first")
	}
	if !rep.Vision || !rep.Whisper || !min.Vision || !min.Whisper {
		t.Fatalf("all packs get vision+whisper")
	}
}

func TestPinnedIDs(t *testing.T) {
	m := pinnedIDs()
	if !m["보고서"] || !m["회의록"] || !m["번역"] || m["해킹"] {
		t.Fatalf("%v", m)
	}
}

func TestFetchPackRejectsUnknown(t *testing.T) {
	if err := fetchPack("없는팩"); err == nil {
		t.Fatal("expected reject")
	}
}
