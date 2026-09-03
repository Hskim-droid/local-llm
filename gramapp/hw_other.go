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
	if runtime.GOOS == "darwin" {
		if g := darwinAvailGB(); g > 0 {
			return g
		}
	}
	return 8
}

func darwinAvailGB() float64 {
	page := 16384.0
	if out, err := exec.Command("sysctl", "-n", "hw.pagesize").Output(); err == nil {
		if n, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil && n > 0 {
			page = n
		}
	}
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0
	}
	var pages float64
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		ok := strings.HasPrefix(ln, "Pages free:") ||
			strings.HasPrefix(ln, "Pages inactive:") ||
			strings.HasPrefix(ln, "Pages speculative:") ||
			strings.HasPrefix(ln, "Pages purgeable:")
		if !ok {
			continue
		}
		f := strings.Fields(ln)
		if len(f) < 3 {
			continue
		}
		n := strings.TrimSuffix(f[len(f)-1], ".")
		v, err := strconv.ParseFloat(n, 64)
		if err == nil {
			pages += v
		}
	}
	if pages <= 0 {
		return 0
	}
	return pages * page / (1024 * 1024 * 1024)
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
