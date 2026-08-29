package main

import (
	"strconv"
	"strings"
)

type verifyReport struct {
	Harvested int      `json:"harvested"`
	Kept      int      `json:"kept"`
	Stripped  int      `json:"stripped"`
	Invented  []string `json:"invented"`
}

func unverifiedMark() string { return T("out.check") }

func verifyMeta(meta map[string]any, h harvest) (map[string]any, verifyReport) {
	rep := verifyReport{Harvested: len(h.Raw)}
	seenInv := map[string]bool{}
	patch := func(s string) string {
		out, inv, kept := patchNumbers(s, h)
		rep.Kept += kept
		rep.Stripped += len(inv)
		for _, x := range inv {
			if !seenInv[x] {
				seenInv[x] = true
				rep.Invented = append(rep.Invented, x)
			}
		}
		return out
	}
	if meta == nil {
		meta = map[string]any{}
	}
	out := walkStrings(meta, patch)
	m, _ := out.(map[string]any)
	if m == nil {
		m = meta
	}
	return dropEmptyFacts(m), rep
}

func patchNumbers(s string, h harvest) (string, []string, int) {
	if strings.TrimSpace(s) == "" {
		return s, nil, 0
	}
	toks := findNums(s)
	var invented []string
	kept := 0
	type cut struct {
		start, end int
		ok         bool
		raw        string
	}
	var cuts []cut
	for _, tok := range toks {
		if skipCheck(tok, s) {
			continue
		}
		ok := allowedTok(tok, h)
		cuts = append(cuts, cut{start: tok.Start, end: tok.End, ok: ok, raw: tok.Raw})
		if ok {
			kept++
		} else {
			invented = append(invented, tok.Raw)
		}
	}
	if len(invented) == 0 {
		return s, nil, kept
	}
	for i := len(cuts) - 1; i >= 0; i-- {
		c := cuts[i]
		if c.ok {
			continue
		}
		s = s[:c.start] + unverifiedMark() + s[c.end:]
	}
	return s, invented, kept
}

func skipCheck(tok numTok, whole string) bool {
	if tok.Core == "" {
		return true
	}
	if reHead.MatchString(whole) && tok.Start == 0 {
		return true
	}
	if !tok.Unit && !tok.Time && !strings.Contains(tok.Raw, ",") {
		n, err := strconv.Atoi(strings.Split(tok.Core, ".")[0])
		if err == nil && n >= 0 && n <= 9 {
			return true
		}
	}
	return false
}

func allowedTok(tok numTok, h harvest) bool {
	if h.raws[foldNum(tok.Raw)] {
		return true
	}
	if tok.Core != "" && h.cores[tok.Core] {
		return true
	}
	if tok.Time && h.hours[tok.Core] {
		return true
	}
	if strings.Contains(tok.Raw, "시") && h.hours[tok.Core] {
		return true
	}
	return false
}

func walkStrings(v any, fn func(string) string) any {
	switch t := v.(type) {
	case string:
		return fn(t)
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = walkStrings(x, fn)
		}
		return out
	case []string:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = fn(x)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			out[k] = walkStrings(x, fn)
		}
		return out
	default:
		return v
	}
}

func dropEmptyFacts(meta map[string]any) map[string]any {
	clean := func(v any) []any {
		ss, ok := asStringSlice(v)
		if !ok {
			return nil
		}
		var out []any
		for _, s := range ss {
			t := strings.TrimSpace(s)
			if t == "" || t == unverifiedMark() {
				continue
			}
			out = append(out, s)
		}
		return out
	}
	if x := clean(meta["exec"]); x != nil {
		meta["exec"] = x
	}
	if x := clean(meta["actions"]); x != nil {
		meta["actions"] = x
	}
	if raw, ok := meta["sections"].([]any); ok {
		var secs []any
		for _, x := range raw {
			m, _ := x.(map[string]any)
			if m == nil {
				continue
			}
			if f := clean(m["facts"]); f != nil {
				m["facts"] = f
			}
			secs = append(secs, m)
		}
		meta["sections"] = secs
	}
	return meta
}

func scoreOutput(source, output string) (matched, invented []string) {
	h := harvestText(source)
	for _, tok := range findNums(output) {
		if skipCheck(tok, output) {
			continue
		}
		if allowedTok(tok, h) {
			matched = append(matched, tok.Raw)
		} else {
			invented = append(invented, tok.Raw)
		}
	}
	return
}
