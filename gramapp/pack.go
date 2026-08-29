package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed packs
var embeddedPacks embed.FS

type pack struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Blurb       string   `json:"blurb"`
	TitleSuffix string   `json:"title_suffix"`
	OutSuffix   string   `json:"out_suffix"`
	OutName     string   `json:"out_name"`
	BodyHeading string   `json:"body_heading"`
	Inputs      []string `json:"inputs"`
	Vision      bool     `json:"vision"`
	Whisper     bool     `json:"whisper"`
	ChunkSys    string
	AssembleSys string
}

func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Dir(p)
}

func loadPacks() ([]pack, error) {
	var merged []pack
	seen := map[string]bool{}
	add := func(ps []pack) {
		for _, p := range ps {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			merged = append(merged, p)
		}
	}
	if ps, err := loadPacksFromDir(filepath.Join(exeDir(), "packs")); err == nil {
		add(ps)
	}
	if ps, err := loadPacksFromDir(filepath.Join(toolsDir(), "packs")); err == nil {
		add(ps)
	}
	if ps, err := loadPacksFromDir("packs"); err == nil {
		add(ps)
	}
	if ps, err := loadPacksFromFS(embeddedPacks, "packs"); err == nil {
		add(ps)
	}
	sortPacks(merged)
	if len(merged) == 0 {
		return nil, fmt.Errorf("양식 팩을 못 찾았습니다")
	}
	return merged, nil
}

func loadPacksFromDir(root string) ([]pack, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []pack
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		p, err := readPackFiles(func(name string) ([]byte, error) {
			return os.ReadFile(filepath.Join(root, e.Name(), name))
		}, e.Name())
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	sortPacks(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("empty packs dir")
	}
	return out, nil
}

func loadPacksFromFS(fsys fs.FS, root string) ([]pack, error) {
	ents, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}
	var out []pack
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		base := root + "/" + e.Name()
		p, err := readPackFiles(func(name string) ([]byte, error) {
			return fs.ReadFile(fsys, base+"/"+name)
		}, e.Name())
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	sortPacks(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no embedded packs")
	}
	return out, nil
}

func readPackFiles(read func(string) ([]byte, error), fallbackID string) (pack, error) {
	raw, err := read("pack.json")
	if err != nil {
		return pack{}, err
	}
	var p pack
	if err := json.Unmarshal(raw, &p); err != nil {
		return pack{}, err
	}
	if p.ID == "" {
		p.ID = fallbackID
	}
	if p.Label == "" {
		p.Label = p.ID
	}
	if p.OutSuffix == "" {
		p.OutSuffix = "_" + p.ID
	}
	if p.OutName == "" {
		p.OutName = p.ID + ".docx"
	}
	if p.TitleSuffix == "" {
		p.TitleSuffix = p.ID
	}
	if p.BodyHeading == "" {
		p.BodyHeading = "내용"
	}
	tightenPack(&p)
	ch, err := read("chunk.txt")
	if err != nil {
		return pack{}, err
	}
	as, err := read("assemble.txt")
	if err != nil {
		return pack{}, err
	}
	p.ChunkSys = strings.TrimSpace(string(ch))
	p.AssembleSys = strings.TrimSpace(string(as))
	if p.ChunkSys == "" || p.AssembleSys == "" {
		return pack{}, fmt.Errorf("empty prompts in %s", p.ID)
	}
	return p, nil
}

func sortPacks(ps []pack) {
	order := map[string]int{"보고서": 1, "회의록": 2, "번역": 3}
	sort.Slice(ps, func(i, j int) bool {
		a, okA := order[ps[i].ID]
		b, okB := order[ps[j].ID]
		if okA && okB {
			return a < b
		}
		if okA {
			return true
		}
		if okB {
			return false
		}
		return ps[i].ID < ps[j].ID
	})
}

func pickPack(arg string) (pack, error) {
	packs, err := catalogPacks()
	if err != nil || len(packs) == 0 {
		return pack{}, fmt.Errorf("양식 팩을 못 찾았습니다")
	}
	if arg != "" {
		if p, ok := matchPack(packs, arg); ok {
			return ensurePackLoaded(p)
		}
		return pack{}, fmt.Errorf("알 수 없는 용도: " + arg)
	}
	sayCanDo(packs)
	choice := strings.TrimSpace(promptLine("> "))
	if isQuit(choice) {
		return pack{}, fmt.Errorf("종료")
	}
	if choice == "" {
		choice = "1"
	}
	if p, ok := matchPack(packs, choice); ok {
		return ensurePackLoaded(p)
	}
	say("지금은 목록에 있는 것만 할 수 있습니다.")
	return ensurePackLoaded(packs[0])
}

var allInputs = []string{
	"pptx", "pdf", "txt", "md", "csv", "tsv", "json", "html", "htm", "xml",
	"docx", "xlsx", "hwpx",
	"mp4", "m4a", "mov", "wav", "mp3", "webm",
}

func tightenPack(p *pack) {
	p.Inputs = append([]string{}, allInputs...)
	p.Vision = true
	p.Whisper = true
}

func packAccepts(_ pack, path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	for _, a := range allInputs {
		if a == ext {
			return true
		}
	}
	return false
}

func isQuit(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "끝", "종료", "q", "quit", "exit":
		return true
	default:
		return false
	}
}

func sayCanDo(packs []pack) {
	var names []string
	for _, p := range packs {
		names = append(names, p.Label)
	}
	say("")
	say("지금은 " + strings.Join(names, ", ") + "을 할 수 있습니다.")
	for i, p := range packs {
		say(fmt.Sprintf("  %d) %s  — %s", i+1, p.Label, p.Blurb))
	}
	say("  끝 이라고 쓰면 종료합니다.")
}

func matchPack(packs []pack, s string) (pack, bool) {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(packs) {
		return packs[n-1], true
	}
	for _, p := range packs {
		if p.ID == s || p.Label == s {
			return p, true
		}
	}
	return pack{}, false
}
