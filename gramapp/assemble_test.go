package main

import "testing"

func TestAssembleThinFallback(t *testing.T) {
	setUILang("ko")
	if !assembleThin(map[string]any{"title": "senolytic", "exec": []any{"원문 정리"}}) {
		t.Fatal("want thin")
	}
	if assembleThin(map[string]any{
		"exec": []any{"약 소개"},
		"sections": []any{
			map[string]any{"heading": "1", "facts": []any{"a", "b"}},
		},
	}) {
		t.Fatal("want kept")
	}
}

func TestFillThinAssembleUsesMerge(t *testing.T) {
	setUILang("ko")
	b := mergeFacts([]rawFact{
		{Text: "TM5614는 경구 약물이다", Location: "a.pptx/1"},
		{Text: "임상 13건이 진행 중이다", Location: "a.pptx/2"},
		{Text: "희귀의약품으로 지정됐다", Location: "a.pptx/3"},
		{Text: "응답률 24.1%이다", Location: "a.pptx/4"},
	}, 1)
	meta := fillThinAssemble(map[string]any{"title": "senolytic", "exec": []any{"원문 정리"}}, "senolytic", b, harvest{Raw: []string{"13", "24.1%"}})
	if assembleThin(meta) {
		t.Fatalf("still thin %#v", meta)
	}
	exec, _ := asStringSlice(meta["exec"])
	if len(exec) < 3 || exec[0] == "원문 정리" {
		t.Fatalf("exec %#v", exec)
	}
	raw, _ := meta["sections"].([]any)
	if len(raw) == 0 {
		t.Fatal("no sections")
	}
}

func TestCleanFactLineStripsMissing(t *testing.T) {
	setUILang("ko")
	got := cleanFactLine("효과 있다. ((원문에 없음))")
	if got != "효과 있다." {
		t.Fatal(got)
	}
}
