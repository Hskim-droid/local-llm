package main

import "testing"

func TestParseJSONObjStripsThink(t *testing.T) {
	m, err := parseJSONObj("</think>\n{\"title\":\"안녕\"}")
	if err != nil || m["title"] != "안녕" {
		t.Fatalf("%v %#v", err, m)
	}
}

func TestChatSpecsGram16Only8B(t *testing.T) {
	p := profile{ID: "gram16"}
	s := chatSpecs(p)
	if len(s) != 1 || s[0].ID != "qwen3-8b" {
		t.Fatalf("%#v", s)
	}
}

func TestSpecByID(t *testing.T) {
	g, ok := specByID("qwen3-14b")
	if !ok || g.File != "Qwen3-14B-Q4_K_M.gguf" {
		t.Fatalf("%v %#v", ok, g)
	}
}
