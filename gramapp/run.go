package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

const sysChunk = "너는 회의록 정리기다. 원문 사실만 한국어 개조식 JSON으로. 창작 금지. 숫자는 원문 그대로. 명사형 종결. 일본어·영어는 한국어로 직역. JSON만."
const sysPack = "너는 회의록 편집기다. 사실만 주제별로 묶는다. 슬라이드마다 섹션을 만들지 말 것. 없는 내용 금지. 명사형 종결. JSON만."

func runFiles(files []string, p profile, model string) (string, error) {
	var segs []segment
	for _, f := range files {
		ss, err := extractPath(f)
		if err != nil {
			say("  건너뜀 " + filepath.Base(f) + " — " + err.Error())
			continue
		}
		base := filepath.Base(f)
		for _, s := range ss {
			s.Location = base + "/" + s.Location
			segs = append(segs, s)
		}
	}
	if len(segs) == 0 {
		return "", fmt.Errorf("꺼낼 글이 없습니다")
	}
	say(fmt.Sprintf("추출 %d개 파일 · 세그먼트 %d", len(files), len(segs)))

	var facts []map[string]any
	buf := ""
	loc := ""
	flush := func() {
		if strings.TrimSpace(buf) == "" {
			return
		}
		user := fmt.Sprintf("위치:%s\n다음 원문을 사실 JSON으로.\n{\"heading_hint\":\"\",\"facts\":[\"명사형\"],\"rows\":[]}\n\n%s", loc, buf)
		obj, err := ollamaChatJSON(model, sysChunk, user, p.Ctx, p.Predict)
		if err != nil {
			obj = map[string]any{"heading_hint": loc, "facts": []any{buf}}
		}
		facts = append(facts, obj)
		buf, loc = "", ""
	}
	for _, s := range segs {
		if loc == "" {
			loc = s.Location
		}
		if len(buf)+len(s.Text) > p.Chunk && buf != "" {
			flush()
			loc = s.Location
		}
		if buf != "" {
			buf += "\n\n"
		}
		buf += "[" + s.Location + "]\n" + s.Text
	}
	flush()

	title := strings.TrimSuffix(filepath.Base(files[0]), filepath.Ext(files[0]))
	packUser := fmt.Sprintf("제목후보:%s\n사실만 주제별 회의록 JSON.\n{\"title\":\"\",\"date\":\"\",\"time\":\"\",\"place\":\"\",\"attendees\":\"\",\"agenda\":\"\",\"exec\":[\"\"],\"sections\":[{\"heading\":\"1. 개요\",\"facts\":[\"\"]}],\"actions\":[\"\"]}\n\n%v", title, facts)
	meta, err := ollamaChatJSON(model, sysPack, packUser, p.Ctx, p.Predict)
	if err != nil {
		meta = map[string]any{"title": title, "exec": []any{"원문 정리"}, "actions": []any{"추가 확인 필요"}}
	}

	outDir := filepath.Join(filepath.Dir(files[0]), stemName(files[0])+"_보고서")
	if err := ensureDir(outDir); err != nil {
		return "", err
	}
	docx := filepath.Join(outDir, "보고서.docx")
	titleOut, _ := meta["title"].(string)
	if titleOut == "" {
		titleOut = title + " 회의록"
	}
	header := [][]string{
		{"일 시", str(meta["date"]), "회의시간", str(meta["time"])},
		{"장 소", str(meta["place"])},
		{"참석자", str(meta["attendees"])},
		{"주요 아젠다", str(meta["agenda"])},
	}
	var sections []map[string]any
	if e, ok := asStringSlice(meta["exec"]); ok && len(e) > 0 {
		sections = append(sections, map[string]any{
			"heading": "0. Executive Summary",
			"blocks":  []map[string]any{{"bullets": e}},
		})
	}
	if raw, ok := meta["sections"].([]any); ok {
		for _, x := range raw {
			m, _ := x.(map[string]any)
			if m == nil {
				continue
			}
			bl := []map[string]any{}
			if facts, ok := asStringSlice(m["facts"]); ok {
				bl = append(bl, map[string]any{"bullets": facts})
			}
			sections = append(sections, map[string]any{"heading": str(m["heading"]), "blocks": bl})
		}
	}
	if acts, ok := asStringSlice(meta["actions"]); ok && len(acts) > 0 {
		sections = append(sections, map[string]any{
			"heading": "향후 계획",
			"blocks":  []map[string]any{{"ordered": acts}},
		})
	}
	if err := writeDocx(docx, titleOut, header, sections); err != nil {
		return "", err
	}
	dumpJSON(filepath.Join(outDir, "보고서.json"), meta)
	return docx, nil
}

func stemName(p string) string {
	b := filepath.Base(p)
	return strings.TrimSuffix(b, filepath.Ext(b))
}

func str(v any) string {
	s, _ := v.(string)
	if strings.TrimSpace(s) == "" {
		return "(원문에 없음)"
	}
	return s
}

func ensureDir(p string) error {
	return mkdirAll(p)
}
