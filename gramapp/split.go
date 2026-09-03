package main

import (
	"fmt"
	"strings"
)

func splitLongSegments(segs []segment, max int) []segment {
	if max < 400 {
		max = 400
	}
	var out []segment
	for _, s := range segs {
		parts := splitLongText(s.Text, max)
		if len(parts) <= 1 {
			out = append(out, s)
			continue
		}
		for i, p := range parts {
			ns := s
			ns.Text = p
			ns.Location = fmt.Sprintf("%s#%d", s.Location, i+1)
			out = append(out, ns)
		}
	}
	return out
}

func splitLongText(s string, max int) []string {
	s = strings.TrimSpace(s)
	if max < 200 {
		max = 200
	}
	if len(s) <= max {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	var parts []string
	for len(s) > max {
		window := s[:max]
		cut := max
		if i := strings.LastIndex(window, "\n\n"); i > max/3 {
			cut = i
		} else if i := strings.LastIndexAny(window, "\n.。"); i > max/3 {
			cut = i + 1
		}
		part := strings.TrimSpace(s[:cut])
		if part != "" {
			parts = append(parts, part)
		}
		s = strings.TrimSpace(s[cut:])
	}
	if s != "" {
		parts = append(parts, s)
	}
	return parts
}
