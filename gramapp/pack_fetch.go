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
			return nil, fmt.Errorf("양식 팩을 못 찾았습니다")
		}
		return local, nil
	}
	seen := map[string]bool{}
	for _, p := range local {
		seen[p.ID] = true
	}
	for _, r := range remote {
		if seen[r.ID] {
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

func fetchPack(id string) error {
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
	say("  팩을 공개 저장소에서 받습니다: " + p.ID)
	if err := fetchPack(p.ID); err != nil {
		return p, fmt.Errorf("팩 받기 실패: %v", err)
	}
	all, err := loadPacks()
	if err != nil {
		return p, err
	}
	if g, ok := matchPack(all, p.ID); ok && g.ChunkSys != "" {
		return g, nil
	}
	return p, fmt.Errorf("받은 팩을 못 읽었습니다: %s", p.ID)
}
