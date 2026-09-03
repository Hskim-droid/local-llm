package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

//go:embed machine.json
var embeddedMachine []byte

const machineRawURL = "https://raw.githubusercontent.com/Hskim-droid/local-llm/main/gramapp/machine.json"

type machineFile struct {
	Version  int              `json:"version"`
	LlamaRel string           `json:"llama_rel"`
	Quant    string           `json:"quant"`
	Ngl      map[string]int   `json:"ngl"`
	Profiles []machineProfile `json:"profiles"`
	Derate   []derateRule     `json:"derate"`
}

type machineProfile struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	OS         string   `json:"os"`
	RamMin     float64  `json:"ram_min"`
	RamMax     float64  `json:"ram_max"`
	AvailMin   float64  `json:"avail_min"`
	Pull       []string `json:"pull"`
	Ctx        int      `json:"ctx"`
	Chunk      int      `json:"chunk"`
	Predict    int      `json:"predict"`
	ThreadsCap int      `json:"threads_cap"`
}

type derateRule struct {
	AvailLt    float64 `json:"avail_lt"`
	Force8b    bool    `json:"force_8b"`
	CtxCap     int     `json:"ctx_cap"`
	ChunkCap   int     `json:"chunk_cap"`
	PredictCap int     `json:"predict_cap"`
	ThreadsCap int     `json:"threads_cap"`
}

const machineTTL = 6 * time.Hour

var loadedMachine machineFile

func loadMachineCatalog() {
	loadedMachine = decodeMachine(embeddedMachine)
	if b, err := os.ReadFile(filepath.Join(exeDir(), "machine.json")); err == nil {
		if m := decodeMachine(b); m.Version >= loadedMachine.Version && len(m.Profiles) > 0 {
			loadedMachine = m
		}
	}
	if cache, ok := readMachineCache(); ok {
		if cache.Version >= loadedMachine.Version && len(cache.Profiles) > 0 {
			loadedMachine = cache
		}
		if machineCacheFresh() {
			return
		}
	}
	if remote, err := fetchMachine(); err == nil && remote.Version >= loadedMachine.Version && len(remote.Profiles) > 0 {
		loadedMachine = remote
		_ = os.MkdirAll(filepath.Dir(machineCachePath()), 0755)
		_ = os.WriteFile(machineCachePath(), mustMachineJSON(remote), 0644)
		_ = os.WriteFile(machineStampPath(), []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	}
}

func machineCachePath() string {
	return filepath.Join(toolsDir(), "machine.json")
}

func machineStampPath() string {
	return filepath.Join(toolsDir(), "machine.fetched")
}

func readMachineCache() (machineFile, bool) {
	b, err := os.ReadFile(machineCachePath())
	if err != nil {
		return machineFile{}, false
	}
	m := decodeMachine(b)
	if len(m.Profiles) == 0 {
		return machineFile{}, false
	}
	return m, true
}

func machineCacheFresh() bool {
	b, err := os.ReadFile(machineStampPath())
	if err != nil {
		st, e := os.Stat(machineCachePath())
		return e == nil && time.Since(st.ModTime()) < machineTTL
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	return err == nil && time.Since(t) < machineTTL
}

func decodeMachine(b []byte) machineFile {
	var m machineFile
	if json.Unmarshal(b, &m) != nil {
		return machineFile{}
	}
	sort.Slice(m.Derate, func(i, j int) bool { return m.Derate[i].AvailLt < m.Derate[j].AvailLt })
	return m
}

var pin8B bool

func liveProfile() profile {
	p := pickProfileHost(snapshotHost())
	if pin8B {
		p.Pull = []string{"qwen3-8b"}
	}
	return p
}

func mustMachineJSON(m machineFile) []byte {
	b, _ := json.MarshalIndent(m, "", "  ")
	return b
}

func fetchMachine() (machineFile, error) {
	cli := &http.Client{Timeout: 12 * time.Second}
	resp, err := cli.Get(machineRawURL)
	if err != nil {
		return machineFile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return machineFile{}, fmt.Errorf("machine.json %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return machineFile{}, err
	}
	m := decodeMachine(b)
	if len(m.Profiles) == 0 {
		return machineFile{}, fmt.Errorf("empty machine.json")
	}
	return m, nil
}

func llamaRelease() string {
	if loadedMachine.LlamaRel != "" {
		return loadedMachine.LlamaRel
	}
	return llamaRel
}

func nglForOS() int {
	if loadedMachine.Ngl != nil {
		if n, ok := loadedMachine.Ngl[runtime.GOOS]; ok {
			return n
		}
	}
	if runtime.GOOS == "darwin" {
		return 99
	}
	return 0
}

func matchMachineProfile(gb float64, goos string, rows []machineProfile) machineProfile {
	return matchHostProfile(hostSnap{Total: gb, Avail: 1e9, OS: goos}, rows)
}

func matchHostProfile(h hostSnap, rows []machineProfile) machineProfile {
	for _, r := range rows {
		if r.OS != "" && r.OS != h.OS {
			continue
		}
		if h.Total < r.RamMin {
			continue
		}
		if r.RamMax > 0 && h.Total >= r.RamMax {
			continue
		}
		if r.AvailMin > 0 && h.Avail < r.AvailMin {
			continue
		}
		return r
	}
	if len(rows) > 0 {
		return rows[len(rows)-1]
	}
	return machineProfile{}
}

func applyDerate(p profile, h hostSnap, rules []derateRule) profile {
	for _, d := range rules {
		if d.AvailLt <= 0 || h.Avail >= d.AvailLt {
			continue
		}
		if d.Force8b {
			p.Pull = []string{"qwen3-8b"}
		}
		if d.CtxCap > 0 && p.Ctx > d.CtxCap {
			p.Ctx = d.CtxCap
		}
		if d.ChunkCap > 0 && p.Chunk > d.ChunkCap {
			p.Chunk = d.ChunkCap
		}
		if d.PredictCap > 0 && p.Predict > d.PredictCap {
			p.Predict = d.PredictCap
		}
		if d.ThreadsCap > 0 && (p.ThreadsCap == 0 || p.ThreadsCap > d.ThreadsCap) {
			p.ThreadsCap = d.ThreadsCap
		}
		return p
	}
	return p
}

func profileFromMachine(r machineProfile) profile {
	if r.ID == "" {
		return profile{}
	}
	return profile{
		ID:         r.ID,
		Label:      r.Label,
		Pull:       r.Pull,
		Ctx:        r.Ctx,
		Chunk:      r.Chunk,
		Predict:    r.Predict,
		ThreadsCap: r.ThreadsCap,
	}
}
