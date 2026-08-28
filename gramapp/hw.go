package main

import (
	"os/exec"
	"runtime"
	"strings"
)

type profile struct {
	ID     string
	Label  string
	Pull   []string
	Ctx    int
	Chunk  int
	Predict int
}

func ramGB() float64 {
	return ramGBOS()
}

func pickProfile(gb float64) profile {
	if runtime.GOOS == "darwin" && gb >= 20 && gb < 28 {
		return profile{ID: "mac24", Label: "맥 24GB", Pull: []string{"qwen3:14b"}, Ctx: 8192, Chunk: 1600, Predict: 1400}
	}
	if gb >= 20 {
		return profile{ID: "gram32", Label: "그램 32GB", Pull: []string{"qwen3:8b", "qwen3:14b"}, Ctx: 6144, Chunk: 1400, Predict: 1100}
	}
	return profile{ID: "gram16", Label: "그램 16GB", Pull: []string{"qwen3:8b"}, Ctx: 4096, Chunk: 1000, Predict: 800}
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
