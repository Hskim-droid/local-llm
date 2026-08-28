package main

import (
	"strings"
	"testing"
)

func TestPatchStripsInventedMoney(t *testing.T) {
	h := harvestText("시드 7억원. 프리머니 약 50억원. 두 곳 4억원 확약.")
	got, inv, kept := patchNumbers("시드 7억원 유치, 후속 80억원 논의함.", h)
	if kept < 1 {
		t.Fatalf("kept 7억원, got kept=%d inv=%v text=%s", kept, inv, got)
	}
	if !strings.Contains(got, unverifiedMark) {
		t.Fatalf("expected mark, got %q", got)
	}
	if strings.Contains(got, "80억원") {
		t.Fatalf("80억원 should be gone: %q", got)
	}
	if !strings.Contains(got, "7억원") {
		t.Fatalf("7억원 should stay: %q", got)
	}
}

func TestVerifyMetaDropsEmptyInventedFact(t *testing.T) {
	h := harvestText("참석자 3명")
	meta := map[string]any{
		"exec": []any{"3명 참석함", "투자 90억원"},
		"sections": []any{
			map[string]any{"heading": "1. 개요", "facts": []any{"90억원 규모"}},
		},
	}
	out, rep := verifyMeta(meta, h)
	if rep.Stripped == 0 {
		t.Fatalf("expected strips, %+v", rep)
	}
	exec, _ := asStringSlice(out["exec"])
	if len(exec) < 1 || !strings.Contains(exec[0], "3명") {
		t.Fatalf("exec=%v", exec)
	}
	blob := strings.Join(exec, " ")
	if strings.Contains(blob, "90억원") {
		t.Fatalf("90억원 should be gone: %v", exec)
	}
}

func TestSectionHeadingNumberNotStripped(t *testing.T) {
	h := harvestText("논의만 있음")
	got, inv, _ := patchNumbers("1. 개요", h)
	if got != "1. 개요" || len(inv) != 0 {
		t.Fatalf("heading stripped: %q inv=%v", got, inv)
	}
}
