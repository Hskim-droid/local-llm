package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func runFiles(files []string, p profile, model string, pk pack) (string, error) {
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
	say(fmt.Sprintf("추출 %d개 파일 · 세그먼트 %d · 팩 %s", len(files), len(segs), pk.ID))
	hv := harvestSegments(segs)
	say(fmt.Sprintf("수확 수치 %d개 (코드). 모델은 이 목록 밖 숫자를 쓰면 검수에서 지웁니다.", len(hv.Raw)))
	allow := harvestPrompt(hv)

	var facts []map[string]any
	buf := ""
	loc := ""
	flush := func() {
		if strings.TrimSpace(buf) == "" {
			return
		}
		user := fmt.Sprintf("위치:%s\n%s\n다음 원문을 사실 JSON으로.\n{\"heading_hint\":\"\",\"facts\":[\"명사형\"],\"rows\":[]}\n\n%s", loc, allow, buf)
		obj, err := ollamaChatJSON(model, pk.ChunkSys, user, p.Ctx, p.Predict)
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
	packUser := fmt.Sprintf("제목후보:%s\n%s\n사실만 주제별 JSON.\n{\"title\":\"\",\"date\":\"\",\"time\":\"\",\"place\":\"\",\"attendees\":\"\",\"agenda\":\"\",\"exec\":[\"\"],\"sections\":[{\"heading\":\"1. 개요\",\"facts\":[\"\"]}],\"actions\":[\"\"]}\n\n%v", title, allow, facts)
	meta, err := ollamaChatJSON(model, pk.AssembleSys, packUser, p.Ctx, p.Predict)
	if err != nil {
		meta = map[string]any{"title": title, "exec": []any{"원문 정리"}, "actions": []any{"추가 확인 필요"}}
	}
	var rep verifyReport
	meta, rep = verifyMeta(meta, hv)
	if rep.Stripped > 0 {
		say(fmt.Sprintf("검수: 원문에 없는 숫자 %d개를 〔원문 확인〕으로 바꿨습니다.", rep.Stripped))
	} else {
		say("검수: 원문에 없는 숫자 없음")
	}

	outDir := filepath.Join(filepath.Dir(files[0]), stemName(files[0])+pk.OutSuffix)
	if err := ensureDir(outDir); err != nil {
		return "", err
	}
	docx := filepath.Join(outDir, pk.OutName)
	titleOut, _ := meta["title"].(string)
	if titleOut == "" {
		titleOut = title + " " + pk.TitleSuffix
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
	if err := writeDocx(docx, titleOut, pk.BodyHeading, header, sections); err != nil {
		return "", err
	}
	stem := strings.TrimSuffix(pk.OutName, filepath.Ext(pk.OutName))
	dumpJSON(filepath.Join(outDir, stem+".json"), meta)
	dumpJSON(filepath.Join(outDir, "수확.json"), map[string]any{"numbers": hv.Raw})
	dumpJSON(filepath.Join(outDir, "검수.json"), rep)
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
