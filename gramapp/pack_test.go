package main

import "testing"

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
