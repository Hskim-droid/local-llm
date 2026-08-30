package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func oneFileJobs(files []string) [][]string {
	out := make([][]string, 0, len(files))
	for _, f := range files {
		out = append(out, []string{f})
	}
	return out
}

func runFiles(files []string, p profile, model string, pk pack) (string, error) {
	var segs []segment
	var imgs []imgPart
	var pending []string
	for _, f := range files {
		if !packAccepts(pk, f) {
			say(T("run.skip.kind", filepath.Base(f)))
			continue
		}
		ss, err := extractPath(f)
		if _, ok := err.(needWhisperError); ok {
			pending = append(pending, f)
			continue
		}
		if err != nil {
			say(T("run.skip", filepath.Base(f), err.Error()))
			continue
		}
		base := filepath.Base(f)
		for _, s := range ss {
			s.Location = base + "/" + s.Location
			segs = append(segs, s)
		}
		for _, im := range extractVisuals(f) {
			im.Location = base + "/" + im.Location
			imgs = append(imgs, im)
		}
	}
	if len(pending) > 0 {
		transcribePending(pending, model)
		for _, f := range pending {
			ss, err := extractPath(f)
			if err != nil {
				say(T("run.skip", filepath.Base(f), err.Error()))
				continue
			}
			base := filepath.Base(f)
			for _, s := range ss {
				s.Location = base + "/" + s.Location
				segs = append(segs, s)
			}
		}
	}
	if len(imgs) > 8 {
		imgs = imgs[:8]
	}
	if len(imgs) > 0 {
		segs = fillVision(segs, imgs)
	}
	if len(segs) == 0 {
		return "", fmt.Errorf("%s", T("run.empty"))
	}
	if err := llamaStart(model, p); err != nil {
		return "", err
	}
	defer llamaStop()
	say(T("run.extract", len(files), len(segs), pk.ID))
	hv := harvestSegments(segs)
	say(T("run.harvest", len(hv.Raw)))
	allow := harvestPrompt(hv)

	var facts []map[string]any
	buf := ""
	loc := ""
	flush := func() {
		if strings.TrimSpace(buf) == "" {
			return
		}
		user := T("llm.chunk", loc, allow, buf)
		obj, err := ollamaChatJSON(model, withLang(pk.ChunkSys), user, p.Ctx, p.Predict)
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

	nFiles := 0
	seenF := map[string]bool{}
	for _, s := range segs {
		f := sourceFile(s.Location)
		if f == "" || seenF[f] {
			continue
		}
		seenF[f] = true
		nFiles++
	}
	engineBundle := mergeFacts(flattenChunkFacts(facts), nFiles)
	say(T("run.merge", len(engineBundle.Shared), len(engineBundle.Unique), len(engineBundle.Conflict)))
	uniqueSidecar := pk.ID != "번역"
	assembleBundle := engineBundle
	limit := p.Chunk * 3
	if limit < 2400 {
		limit = 2400
	}
	if mergePayloadSize(assembleBundle, uniqueSidecar) > limit {
		say(T("run.fold"))
		assembleBundle = compactBundleForAssemble(engineBundle, p, model, allow)
	}

	title := strings.TrimSuffix(filepath.Base(files[0]), filepath.Ext(files[0]))
	payload := formatMergePayload(assembleBundle, uniqueSidecar)
	if strings.TrimSpace(payload) == "" {
		payload = fmt.Sprintf("%v", facts)
	}
	packUser := T("llm.assemble", title, allow, payload)
	meta, err := ollamaChatJSON(model, withLang(pk.AssembleSys), packUser, p.Ctx, p.Predict)
	if err != nil {
		meta = map[string]any{"title": title, "exec": []any{T("out.fallback.exec")}, "actions": []any{T("out.fallback.act")}}
	}
	var rep verifyReport
	meta, rep = verifyMeta(meta, hv)
	if rep.Stripped > 0 {
		say(T("run.verify.n", rep.Stripped))
	} else {
		say(T("run.verify.ok"))
	}

	outSuf, outName, titleSuf, bodyHead := packOutputs(pk)
	outDir := filepath.Join(filepath.Dir(files[0]), stemName(files[0])+outSuf)
	if err := ensureDir(outDir); err != nil {
		return "", err
	}
	docx := filepath.Join(outDir, outName)
	titleOut, _ := meta["title"].(string)
	if titleOut == "" {
		titleOut = title + " " + titleSuf
	}
	header := [][]string{
		{T("out.date"), str(meta["date"]), T("out.time"), str(meta["time"])},
		{T("out.place"), str(meta["place"])},
		{T("out.people"), str(meta["attendees"])},
		{T("out.agenda"), str(meta["agenda"])},
	}
	var sections []map[string]any
	if e, ok := asStringSlice(meta["exec"]); ok && len(e) > 0 {
		sections = append(sections, map[string]any{
			"heading": T("out.exec"),
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
			"heading": T("out.next"),
			"blocks":  []map[string]any{{"ordered": acts}},
		})
	}
	sections = appendMergeSections(sections, engineBundle, uniqueSidecar)
	if err := writeDocx(docx, titleOut, bodyHead, header, sections); err != nil {
		return "", err
	}
	stem := strings.TrimSuffix(outName, filepath.Ext(outName))
	dumpJSON(filepath.Join(outDir, stem+".json"), meta)
	dumpJSON(filepath.Join(outDir, T("out.json.harvest")), map[string]any{"numbers": hv.Raw})
	dumpJSON(filepath.Join(outDir, T("out.json.verify")), rep)
	dumpJSON(filepath.Join(outDir, T("out.json.merge")), bundleDump(engineBundle))
	return docx, nil
}

func stemName(p string) string {
	b := filepath.Base(p)
	return strings.TrimSuffix(b, filepath.Ext(b))
}

func str(v any) string {
	s, _ := v.(string)
	if strings.TrimSpace(s) == "" {
		return T("out.missing")
	}
	return s
}

func ensureDir(p string) error {
	return mkdirAll(p)
}
