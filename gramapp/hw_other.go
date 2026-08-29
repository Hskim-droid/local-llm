//go:build !windows

package main

import (
	"os"
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

func osUILang() string {
	for _, k := range []string{"LC_ALL", "LANG"} {
		if normalizeLang(os.Getenv(k)) == "ko" {
			return "ko"
		}
	}
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
		if err == nil && normalizeLang(string(out)) == "ko" {
			return "ko"
		}
	}
	return "en"
}
