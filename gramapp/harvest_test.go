package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNumberCoreKeepsIntegerZeros(t *testing.T) {
	if numberCore("50억원") != "50" {
		t.Fatalf("50 → %s", numberCore("50억원"))
	}
	if numberCore("5,000億ウォン") != "5000" {
		t.Fatalf("5,000 → %s", numberCore("5,000億ウォン"))
	}
	if numberCore("10.00") != "10" {
		t.Fatalf("10.00 → %s", numberCore("10.00"))
	}
	if numberCore("７억원") != "7" {
		t.Fatalf("fullwidth 7 → %s", numberCore("７억원"))
	}
}

func TestHarvestExamples(t *testing.T) {
	jp := readExample(t, "slides_jp.txt")
	h := harvestText(jp)
	for _, want := range []string{"2026", "1994", "5000", "8", "3", "4"} {
		if !h.cores[want] {
			t.Errorf("slides_jp missing core %s in %v", want, h.Raw)
		}
	}
	if !h.hours["13"] || !h.hours["17"] {
		t.Errorf("slides_jp hours 13/17 missing: %+v raw=%v", h.hours, h.Raw)
	}

	en := readExample(t, "minutes.txt")
	h2 := harvestText(en)
	for _, want := range []string{"2026", "2027", "7", "50", "4"} {
		if !h2.cores[want] {
			t.Errorf("minutes missing core %s in %v", want, h2.Raw)
		}
	}
	if !h2.hours["15"] || !h2.hours["16"] {
		t.Errorf("minutes hours 15/16 missing: raw=%v", h2.Raw)
	}
}

func TestScoreInventedNumber(t *testing.T) {
	src := readExample(t, "minutes.txt")
	ok, bad := scoreOutput(src, "시드 7 billion KRW, 프리머니 50 billion. 후속 80억원 유치함.")
	if len(ok) == 0 {
		t.Fatalf("expected matches, got none")
	}
	found := false
	for _, x := range bad {
		if numberCore(x) == "80" {
			found = true
		}
	}
	if !found {
		t.Fatalf("80억원 should be invented; matched=%v invented=%v", ok, bad)
	}
}

func TestFactoryScoreLogs(t *testing.T) {
	min := readExample(t, "minutes.txt")
	ok, bad := scoreOutput(min, "시드 7 billion KRW, 프리머니 50 billion KRW, 확약 4 billion. 런칭 2027. 없는 80억원.")
	t.Logf("minutes 수확=%d 일치=%d 환각=%d 환각토큰=%v", len(harvestText(min).Raw), len(ok), len(bad), bad)
	if len(bad) == 0 {
		t.Fatal("80억원 should score as invented")
	}

	jp := readExample(t, "slides_jp.txt")
	ok2, bad2 := scoreOutput(jp, "2026년 10월 8일 요코하마. 연결매출 5,000억. 가짜 120명.")
	t.Logf("slides_jp 수확=%d 일치=%d 환각=%d 환각토큰=%v", len(harvestText(jp).Raw), len(ok2), len(bad2), bad2)
	found := false
	for _, x := range bad2 {
		if numberCore(x) == "120" {
			found = true
		}
	}
	if !found {
		t.Fatalf("120명 should be invented; matched=%v invented=%v", ok2, bad2)
	}
}

func readExample(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
