//go:build !windows

package main

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func ramGBOS() float64 {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			n, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
			return n / (1024 * 1024 * 1024)
		}
	}
	return 16
}

func availGB() float64 {
	return 8
}

func setUTF8Console() {}
