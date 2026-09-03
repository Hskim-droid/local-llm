package main

import "strings"

func cleanFactLine(s string) string {
	s = strings.TrimSpace(s)
	for _, tag := range []string{
		" ((" + T("out.missing") + "))",
		" ((원문에 없음))",
		" ((not in source))",
	} {
		s = strings.ReplaceAll(s, tag, "")
	}
	return strings.TrimSpace(s)
}

func factLines(bundle mergeBundle) []string {
	var out []string
	seen := map[string]bool{}
	add := func(items []mergeItem) {
		for _, it := range items {
			ln := cleanFactLine(itemLine(it))
			if ln == "" || seen[ln] {
				continue
			}
			seen[ln] = true
			out = append(out, ln)
		}
	}
	add(bundle.Shared)
	if len(out) == 0 {
		add(bundle.Unique)
	}
	return out
}

func assembleThin(meta map[string]any) bool {
	if meta == nil {
		return true
	}
	n := 0
	if e, ok := asStringSlice(meta["exec"]); ok {
		for _, s := range e {
			s = strings.TrimSpace(s)
			if s == "" || s == T("out.fallback.exec") {
				continue
			}
			n++
		}
	}
	raw, _ := meta["sections"].([]any)
	for _, x := range raw {
		m, _ := x.(map[string]any)
		if m == nil {
			continue
		}
		facts, _ := asStringSlice(m["facts"])
		for _, f := range facts {
			if strings.TrimSpace(f) != "" {
				n++
			}
		}
	}
	return n < 3
}

func fillThinAssemble(meta map[string]any, title string, bundle mergeBundle, hv harvest) map[string]any {
	if meta == nil {
		meta = map[string]any{}
	}
	if !assembleThin(meta) {
		return meta
	}
	lines := factLines(bundle)
	if len(lines) == 0 && len(hv.Raw) == 0 {
		if strings.TrimSpace(str(meta["title"])) == "" {
			meta["title"] = title
		}
		return meta
	}
	execN := 3
	if len(lines) < execN {
		execN = len(lines)
	}
	exec := make([]any, 0, execN)
	for _, s := range lines[:execN] {
		exec = append(exec, s)
	}
	rest := lines[execN:]
	overviewN := 8
	if len(rest) < overviewN {
		overviewN = len(rest)
	}
	overview := make([]any, 0, overviewN)
	for _, s := range rest[:overviewN] {
		overview = append(overview, s)
	}
	core := make([]any, 0)
	for i, s := range rest[overviewN:] {
		if i >= 12 {
			break
		}
		core = append(core, s)
	}
	nums := make([]any, 0, 20)
	for _, n := range hv.Raw {
		if len(nums) >= 20 {
			break
		}
		nums = append(nums, n)
	}
	var sections []any
	if len(overview) > 0 {
		sections = append(sections, map[string]any{"heading": T("skel.h1"), "facts": overview})
	}
	if len(core) > 0 {
		sections = append(sections, map[string]any{"heading": T("skel.h2"), "facts": core})
	}
	if len(nums) > 0 {
		sections = append(sections, map[string]any{"heading": T("skel.h3"), "facts": nums})
	}
	if strings.TrimSpace(str(meta["title"])) == "" {
		meta["title"] = title
	}
	meta["exec"] = exec
	if len(sections) > 0 {
		meta["sections"] = sections
	}
	if acts, ok := asStringSlice(meta["actions"]); !ok || len(acts) == 0 || (len(acts) == 1 && strings.TrimSpace(acts[0]) == T("out.fallback.act")) {
		meta["actions"] = []any{}
	}
	return meta
}
