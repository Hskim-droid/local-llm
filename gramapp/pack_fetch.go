package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const packsRawBase = "https://raw.githubusercontent.com/Hskim-droid/local-llm/main/gramapp/packs"

type packIndex struct {
	Packs []pack `json:"packs"`
}

func catalogPacks() ([]pack, error) {
	local, err := loadPacks()
	if err != nil {
		local = nil
	}
	remote, rerr := fetchPackIndex()
	if rerr != nil || len(remote) == 0 {
		if len(local) == 0 {
			return nil, fmt.Errorf("%s", T("err.pack.none"))
		}
		return local, nil
	}
	seen := map[string]bool{}
	for _, p := range local {
		seen[p.ID] = true
	}
	allow := pinnedIDs()
	for _, r := range remote {
		if seen[r.ID] || !allow[r.ID] {
			continue
		}
		if r.Label == "" {
			r.Label = r.ID
		}
		local = append(local, r)
		seen[r.ID] = true
	}
	sortPacks(local)
	return local, nil
}

func fetchPackIndex() ([]pack, error) {
	cli := &http.Client{Timeout: 12 * time.Second}
	resp, err := cli.Get(packsRawBase + "/index.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("index %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var idx packIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return idx.Packs, nil
}

func packFileURL(id, file string) string {
	return packsRawBase + "/" + url.PathEscape(id) + "/" + file
}

func pinnedIDs() map[string]bool {
	m := map[string]bool{}
	b, err := embeddedPacks.ReadFile("packs/index.json")
	if err != nil {
		return map[string]bool{"보고서": true, "회의록": true, "번역": true}
	}
	var idx packIndex
	if json.Unmarshal(b, &idx) != nil {
		return map[string]bool{"보고서": true, "회의록": true, "번역": true}
	}
	for _, p := range idx.Packs {
		if p.ID != "" {
			m[p.ID] = true
		}
	}
	return m
}

func fetchPack(id string) error {
	if !pinnedIDs()[id] {
		return fmt.Errorf("%s", T("err.pack.fetch"))
	}
	dest := filepath.Join(toolsDir(), "packs", id)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, name := range []string{"pack.json", "chunk.txt", "assemble.txt"} {
		if err := downloadFile(packFileURL(id, name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	return nil
}

func ensurePackLoaded(p pack) (pack, error) {
	if p.ChunkSys != "" && p.AssembleSys != "" {
		return p, nil
	}
	say(T("pack.fetch", p.ID))
	if err := fetchPack(p.ID); err != nil {
		return p, fmt.Errorf("%s", T("err.pack.get", err))
	}
	all, err := loadPacks()
	if err != nil {
		return p, err
	}
	if g, ok := matchPack(all, p.ID); ok && g.ChunkSys != "" {
		return g, nil
	}
	return p, fmt.Errorf("%s", T("err.pack.read", p.ID))
}
