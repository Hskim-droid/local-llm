package main

import (
	"regexp"
	"strings"
)

type harvest struct {
	Raw   []string
	cores map[string]bool
	raws  map[string]bool
	hours map[string]bool
}

type numTok struct {
	Raw   string
	Core  string
	Time  bool
	Unit  bool
	Start int
	End   int
}

var (
	reTime = regexp.MustCompile(`\d{1,2}:\d{2}(?:\s*[-~～〜]\s*\d{1,2}:\d{2})?`)
	reNum  = regexp.MustCompile(`(?i)(?:\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?)(?:\s*(?:억원|만원|천원|달러|엔|원|억|만|천|%|％|명|건|개|년|월|일|시간|분|億ウォン|億|万|円|ウォン|billion|million|thousand|krw|usd|yen|hours?|mins?|people|名|人))?`)
	reCore = regexp.MustCompile(`\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?`)
	reHead = regexp.MustCompile(`^\d+\.\s`)
)

func harvestSegments(segs []segment) harvest {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
		b.WriteByte('\n')
	}
	return harvestText(b.String())
}

func harvestText(s string) harvest {
	h := harvest{
		cores: map[string]bool{},
		raws:  map[string]bool{},
		hours: map[string]bool{},
	}
	seen := map[string]bool{}
	for _, tok := range findNums(s) {
		key := strings.TrimSpace(tok.Raw)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		h.Raw = append(h.Raw, key)
		h.raws[foldNum(key)] = true
		if tok.Time {
			for _, hr := range timeHours(tok.Raw) {
				h.hours[hr] = true
			}
			continue
		}
		if tok.Core != "" {
			h.cores[tok.Core] = true
		}
	}
	return h
}

func findNums(s string) []numTok {
	s = wideDigits(s)
	var out []numTok
	used := make([]bool, len(s))
	mark := func(a, b int) {
		if a < 0 {
			a = 0
		}
		if b > len(s) {
			b = len(s)
		}
		for i := a; i < b; i++ {
			used[i] = true
		}
	}
	overlap := func(a, b int) bool {
		for i := a; i < b && i < len(s); i++ {
			if used[i] {
				return true
			}
		}
		return false
	}
	for _, loc := range reTime.FindAllStringIndex(s, -1) {
		raw := s[loc[0]:loc[1]]
		out = append(out, numTok{Raw: raw, Core: numberCore(raw), Time: true, Unit: true, Start: loc[0], End: loc[1]})
		mark(loc[0], loc[1])
	}
	for _, loc := range reNum.FindAllStringIndex(s, -1) {
		if overlap(loc[0], loc[1]) {
			continue
		}
		raw := strings.TrimSpace(s[loc[0]:loc[1]])
		if raw == "" {
			continue
		}
		out = append(out, numTok{
			Raw:   raw,
			Core:  numberCore(raw),
			Unit:  hasUnit(raw),
			Start: loc[0],
			End:   loc[1],
		})
	}
	return out
}

func numberCore(s string) string {
	s = wideDigits(s)
	m := reCore.FindString(s)
	m = strings.ReplaceAll(m, ",", "")
	if i := strings.Index(m, "."); i >= 0 {
		frac := strings.TrimRight(m[i+1:], "0")
		intp := m[:i]
		if frac == "" {
			return intp
		}
		return intp + "." + frac
	}
	return m
}

func hasUnit(s string) bool {
	low := strings.ToLower(s)
	units := []string{"억", "만", "천", "원", "달러", "엔", "%", "％", "명", "건", "개", "년", "월", "일", "시", "분", "億", "万", "円", "ウォン", "billion", "million", "thousand", "krw", "usd", "yen", "hour", "min", "people", "名", "人", ":"}
	for _, u := range units {
		if strings.Contains(low, strings.ToLower(u)) {
			return true
		}
	}
	return false
}

func timeHours(s string) []string {
	var hs []string
	for _, loc := range regexp.MustCompile(`\d{1,2}:\d{2}`).FindAllString(s, -1) {
		hr := strings.Split(loc, ":")[0]
		hr = strings.TrimLeft(hr, "0")
		if hr == "" {
			hr = "0"
		}
		hs = append(hs, hr)
	}
	return hs
}

func wideDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '０' && r <= '９':
			b.WriteRune('0' + (r - '０'))
		case r == '，' || r == '、':
			b.WriteByte(',')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func foldNum(s string) string {
	s = wideDigits(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return strings.ToLower(s)
}

func harvestPrompt(h harvest) string {
	if len(h.Raw) == 0 {
		return ""
	}
	raw := h.Raw
	if len(raw) > 100 {
		raw = raw[:100]
	}
	return "원문 수치(이 목록에 없는 숫자를 만들지 말 것. 표기는 원문 그대로):\n" + strings.Join(raw, "\n")
}
