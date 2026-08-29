package main

import (
	"strings"
	"testing"
)

func TestNormalizeLang(t *testing.T) {
	if normalizeLang("ko-KR") != "ko" || normalizeLang("Korean") != "ko" {
		t.Fatal("ko")
	}
	if normalizeLang("en_US") != "en" || normalizeLang("english") != "en" {
		t.Fatal("en")
	}
	if normalizeLang("fr") != "" {
		t.Fatal("fr")
	}
}

func TestTEnglishDefault(t *testing.T) {
	setUILang("en")
	if T("cando.only") != "Only items on this list are available right now." {
		t.Fatal(T("cando.only"))
	}
	setUILang("ko")
	if !strings.Contains(T("cando.only"), "목록") {
		t.Fatal(T("cando.only"))
	}
	setUILang("en")
}

func TestResolveLangFlagWins(t *testing.T) {
	t.Setenv("LOCAL_LLM_LANG", "ko")
	if resolveLang("en") != "en" {
		t.Fatal("flag should win")
	}
}

func TestPackAliasMatch(t *testing.T) {
	packs, err := loadPacksFromFS(embeddedPacks, "packs")
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := matchPack(packs, "report"); !ok || p.ID != "보고서" {
		t.Fatalf("report alias")
	}
	if p, ok := matchPack(packs, "minutes"); !ok || p.ID != "회의록" {
		t.Fatalf("minutes alias")
	}
	if p, ok := matchPack(packs, "translation"); !ok || p.ID != "번역" {
		t.Fatalf("translation alias")
	}
}

func TestResolveLangEnv(t *testing.T) {
	t.Setenv("LOCAL_LLM_LANG", "ko")
	if resolveLang("") != "ko" {
		t.Fatal(resolveLang(""))
	}
}
