package main

import (
	"os/exec"
	"runtime"
	"strings"
)

type profile struct {
	ID         string
	Label      string
	Pull       []string
	Ctx        int
	Chunk      int
	Predict    int
	ThreadsCap int
}

func ramGB() float64 {
	return ramGBOS()
}

type hostSnap struct {
	Total float64
	Avail float64
	Load  float64
	OS    string
	Arch  string
	CPUs  int
}

func snapshotHost() hostSnap {
	t := ramGB()
	a := availGB()
	if t > 0 && a > t {
		a = t
	}
	if a < 0 {
		a = 0
	}
	load := 0.0
	if t > 0 {
		load = (1 - a/t) * 100
		if load < 0 {
			load = 0
		}
		if load > 100 {
			load = 100
		}
	}
	return hostSnap{Total: t, Avail: a, Load: load, OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU()}
}

func pickProfile(gb float64) profile {
	return pickProfileHost(hostSnap{Total: gb, Avail: 1e9, OS: runtime.GOOS, CPUs: runtime.NumCPU()})
}

func pickProfileHost(h hostSnap) profile {
	p := profileFromMachine(matchHostProfile(h, loadedMachine.Profiles))
	if p.ID == "" {
		p = pickProfileFallback(h.Total, h.OS)
	}
	return applyDerate(p, h, loadedMachine.Derate)
}

func pickProfileFallback(gb float64, goos string) profile {
	if goos == "darwin" && gb >= 20 && gb < 28 {
		return profile{ID: "mac24", Label: "맥 24GB", Pull: []string{"qwen3-14b"}, Ctx: 8192, Chunk: 1600, Predict: 1400, ThreadsCap: 8}
	}
	if gb >= 20 {
		return profile{ID: "gram32", Label: "그램 32GB", Pull: []string{"qwen3-8b", "qwen3-14b"}, Ctx: 6144, Chunk: 1400, Predict: 1100, ThreadsCap: 8}
	}
	return profile{ID: "gram16", Label: "그램 16GB", Pull: []string{"qwen3-8b"}, Ctx: 4096, Chunk: 1000, Predict: 800, ThreadsCap: 4}
}

func gpuName() string {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("wmic", "path", "win32_VideoController", "get", "Name").Output()
		if err == nil {
			for _, ln := range strings.Split(string(out), "\n") {
				ln = strings.TrimSpace(ln)
				if ln != "" && ln != "Name" {
					return ln
				}
			}
		}
	}
	if runtime.GOOS == "darwin" {
		return "Apple Silicon"
	}
	return "내장 그래픽"
}
