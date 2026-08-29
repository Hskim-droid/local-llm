package main

import (
	"sort"
	"strings"
)

type rawFact struct {
	Text     string
	Location string
}

type mergeSide struct {
	Text    string
	Sources []string
}

type mergeItem struct {
	Kind  string // shared | unique | conflict
	Sides []mergeSide
}

type mergeBundle struct {
	Shared   []mergeItem
	Unique   []mergeItem
	Conflict []mergeItem
	Files    int
}

func flattenChunkFacts(objs []map[string]any) []rawFact {
	var out []rawFact
	for _, o := range objs {
		if o == nil {
			continue
		}
		loc := str(o["heading_hint"])
		facts, ok := asStringSlice(o["facts"])
		if !ok {
			continue
		}
		for _, f := range facts {
			t := strings.TrimSpace(f)
			if t == "" {
				continue
			}
			out = append(out, rawFact{Text: t, Location: loc})
		}
	}
	return out
}

func sourceFile(loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return ""
	}
	if i := strings.Index(loc, "/"); i > 0 {
		return loc[:i]
	}
	return loc
}

func foldFact(s string) string {
	s = wideDigits(strings.TrimSpace(s))
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func factStem(s string) string {
	s = wideDigits(s)
	s = reTime.ReplaceAllString(s, "#")
	s = reCore.ReplaceAllString(s, "#")
	s = strings.ToLower(s)
	var b strings.Builder
	prevSpace := true
	for _, r := range s {
		if r == '#' {
			b.WriteByte('#')
			prevSpace = false
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x80 {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func stemBodyLen(stem string) int {
	n := 0
	for _, r := range stem {
		if r != '#' && r != ' ' {
			n++
		}
	}
	return n
}

func numKey(s string) string {
	seen := map[string]bool{}
	var cores []string
	for _, tok := range findNums(s) {
		if tok.Core == "" || seen[tok.Core] {
			continue
		}
		seen[tok.Core] = true
		cores = append(cores, tok.Core)
	}
	sort.Strings(cores)
	return strings.Join(cores, ",")
}

func uniqSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

type factBucket struct {
	text    string
	fold    string
	stem    string
	nums    string
	sources []string
}

func mergeFacts(facts []rawFact, nFiles int) mergeBundle {
	b := mergeBundle{Files: nFiles}
	if len(facts) == 0 {
		return b
	}
	byFold := map[string]*factBucket{}
	var order []string
	for _, f := range facts {
		fd := foldFact(f.Text)
		if fd == "" {
			continue
		}
		src := sourceFile(f.Location)
		if bk, ok := byFold[fd]; ok {
			bk.sources = append(bk.sources, src)
			if len([]rune(f.Text)) > len([]rune(bk.text)) {
				bk.text = f.Text
			}
			continue
		}
		byFold[fd] = &factBucket{
			text:    f.Text,
			fold:    fd,
			stem:    factStem(f.Text),
			nums:    numKey(f.Text),
			sources: []string{src},
		}
		order = append(order, fd)
	}
	buckets := make([]*factBucket, 0, len(order))
	for _, k := range order {
		bk := byFold[k]
		bk.sources = uniqSorted(bk.sources)
		buckets = append(buckets, bk)
	}
	used := make([]bool, len(buckets))
	for i := 0; i < len(buckets); i++ {
		if used[i] {
			continue
		}
		a := buckets[i]
		for j := i + 1; j < len(buckets); j++ {
			if used[j] {
				continue
			}
			c := buckets[j]
			if a.nums != c.nums {
				continue
			}
			if a.fold == c.fold || strings.Contains(a.fold, c.fold) || strings.Contains(c.fold, a.fold) {
				if len([]rune(c.text)) > len([]rune(a.text)) {
					a.text = c.text
					a.fold = c.fold
					a.stem = c.stem
				}
				a.sources = uniqSorted(append(a.sources, c.sources...))
				used[j] = true
			}
		}
	}
	type stemGroup struct {
		idxs []int
	}
	var groups []stemGroup
	stemAt := map[string]int{}
	for i, bk := range buckets {
		if used[i] {
			continue
		}
		st := bk.stem
		if st == "" || stemBodyLen(st) < 4 {
			groups = append(groups, stemGroup{idxs: []int{i}})
			continue
		}
		if g, ok := stemAt[st]; ok {
			groups[g].idxs = append(groups[g].idxs, i)
			continue
		}
		stemAt[st] = len(groups)
		groups = append(groups, stemGroup{idxs: []int{i}})
	}
	for _, g := range groups {
		numSets := map[string][]int{}
		var numOrder []string
		for _, i := range g.idxs {
			k := buckets[i].nums
			if _, ok := numSets[k]; !ok {
				numOrder = append(numOrder, k)
			}
			numSets[k] = append(numSets[k], i)
		}
		if len(numOrder) >= 2 && stemBodyLen(buckets[g.idxs[0]].stem) >= 4 {
			it := mergeItem{Kind: "conflict"}
			for _, nk := range numOrder {
				side := mergeSide{}
				var src []string
				for _, i := range numSets[nk] {
					if side.Text == "" || len([]rune(buckets[i].text)) > len([]rune(side.Text)) {
						side.Text = buckets[i].text
					}
					src = append(src, buckets[i].sources...)
				}
				side.Sources = uniqSorted(src)
				it.Sides = append(it.Sides, side)
			}
			b.Conflict = append(b.Conflict, it)
			continue
		}
		for _, nk := range numOrder {
			side := mergeSide{}
			var src []string
			for _, i := range numSets[nk] {
				if side.Text == "" || len([]rune(buckets[i].text)) > len([]rune(side.Text)) {
					side.Text = buckets[i].text
				}
				src = append(src, buckets[i].sources...)
			}
			side.Sources = uniqSorted(src)
			it := mergeItem{Sides: []mergeSide{side}}
			multi := nFiles >= 2 && len(side.Sources) >= 2
			if multi {
				it.Kind = "shared"
				b.Shared = append(b.Shared, it)
			} else if nFiles >= 2 {
				it.Kind = "unique"
				b.Unique = append(b.Unique, it)
			} else {
				it.Kind = "shared"
				b.Shared = append(b.Shared, it)
			}
		}
	}
	return b
}

func itemLine(it mergeItem) string {
	var parts []string
	for _, s := range it.Sides {
		src := strings.Join(s.Sources, ", ")
		if src == "" {
			parts = append(parts, s.Text)
			continue
		}
		parts = append(parts, s.Text+" ("+src+")")
	}
	if it.Kind == "conflict" {
		return strings.Join(parts, " ≠ ")
	}
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func writeFactLines(b *strings.Builder, title string, items []mergeItem) {
	if len(items) == 0 {
		return
	}
	b.WriteString(title)
	b.WriteByte('\n')
	for _, it := range items {
		ln := itemLine(it)
		if ln == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func formatMergePayload(bundle mergeBundle, uniqueSidecar bool) string {
	var b strings.Builder
	b.WriteString(T("merge.intro"))
	if uniqueSidecar {
		b.WriteString(T("merge.side.unique"))
	} else {
		b.WriteString(T("merge.side.body"))
	}
	writeFactLines(&b, T("merge.shared"), bundle.Shared)
	if uniqueSidecar {
		if len(bundle.Unique) > 0 {
			b.WriteString(T("merge.unique.skip"))
		}
	} else {
		writeFactLines(&b, T("merge.unique"), bundle.Unique)
	}
	if len(bundle.Conflict) > 0 {
		b.WriteString(T("merge.conflict"))
		for _, it := range bundle.Conflict {
			b.WriteString("- ")
			b.WriteString(itemLine(it))
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}

func mergePayloadSize(bundle mergeBundle, uniqueSidecar bool) int {
	return len(formatMergePayload(bundle, uniqueSidecar))
}

func compactFactLines(lines []string, p profile, model, allow string) []string {
	if len(lines) == 0 {
		return nil
	}
	limit := p.Chunk
	if limit < 400 {
		limit = 400
	}
	var out []string
	var buf []string
	n := 0
	flush := func() {
		if len(buf) == 0 {
			return
		}
		user := T("llm.fold.user", allow, strings.Join(buf, "\n"))
		obj, err := ollamaChatJSON(model, withLang(T("llm.fold")), user, p.Ctx, p.Predict)
		if err != nil {
			out = append(out, buf...)
			buf, n = nil, 0
			return
		}
		got, ok := asStringSlice(obj["facts"])
		if !ok || len(got) == 0 {
			out = append(out, buf...)
		} else {
			for _, s := range got {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		buf, n = nil, 0
	}
	for _, ln := range lines {
		if n+len(ln) > limit && len(buf) > 0 {
			flush()
		}
		buf = append(buf, ln)
		n += len(ln) + 1
	}
	flush()
	return out
}

func linesOf(items []mergeItem) []string {
	var out []string
	for _, it := range items {
		if len(it.Sides) == 0 {
			continue
		}
		out = append(out, it.Sides[0].Text)
	}
	return out
}

func replaceTexts(items []mergeItem, texts []string, kind string) []mergeItem {
	if len(texts) == 0 {
		return items
	}
	out := make([]mergeItem, 0, len(texts))
	for _, t := range texts {
		out = append(out, mergeItem{
			Kind:  kind,
			Sides: []mergeSide{{Text: t}},
		})
	}
	return out
}

func compactBundleForAssemble(bundle mergeBundle, p profile, model, allow string) mergeBundle {
	out := bundle
	out.Shared = replaceTexts(bundle.Shared, compactFactLines(linesOf(bundle.Shared), p, model, allow), "shared")
	out.Unique = replaceTexts(bundle.Unique, compactFactLines(linesOf(bundle.Unique), p, model, allow), "unique")
	return out
}

func appendMergeSections(sections []map[string]any, bundle mergeBundle, uniqueSidecar bool) []map[string]any {
	if uniqueSidecar && len(bundle.Unique) > 0 {
		var bullets []string
		for _, it := range bundle.Unique {
			if ln := itemLine(it); ln != "" {
				bullets = append(bullets, ln)
			}
		}
		if len(bullets) > 0 {
			sections = append(sections, map[string]any{
				"heading": T("out.unique"),
				"blocks":  []map[string]any{{"bullets": bullets}},
			})
		}
	}
	if len(bundle.Conflict) > 0 {
		var bullets []string
		for _, it := range bundle.Conflict {
			if ln := itemLine(it); ln != "" {
				bullets = append(bullets, ln)
			}
		}
		if len(bullets) > 0 {
			sections = append(sections, map[string]any{
				"heading": T("out.conflict"),
				"blocks":  []map[string]any{{"bullets": bullets}},
			})
		}
	}
	return sections
}

func bundleDump(bundle mergeBundle) map[string]any {
	lines := func(items []mergeItem) []string {
		var out []string
		for _, it := range items {
			out = append(out, itemLine(it))
		}
		return out
	}
	return map[string]any{
		"files":    bundle.Files,
		"shared":   lines(bundle.Shared),
		"unique":   lines(bundle.Unique),
		"conflict": lines(bundle.Conflict),
	}
}

func countInputFiles(files []string) int {
	seen := map[string]bool{}
	n := 0
	for _, f := range files {
		base := strings.ToLower(strings.TrimSpace(f))
		if i := strings.LastIndexAny(base, `/\`); i >= 0 {
			base = base[i+1:]
		}
		if base == "" || seen[base] {
			continue
		}
		seen[base] = true
		n++
	}
	if n == 0 {
		return len(files)
	}
	return n
}
