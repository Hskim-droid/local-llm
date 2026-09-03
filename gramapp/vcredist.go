package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const vcRedistURL = "https://aka.ms/vs/17/release/vc_redist.x64.exe"

func ensureVCRedist() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	dll := filepath.Join(root, "System32", "vcruntime140.dll")
	if _, err := os.Stat(dll); err == nil {
		return nil
	}
	say(T("step.vcredist"))
	dest := filepath.Join(toolsDir(), "vc_redist.x64.exe")
	if err := downloadFile(vcRedistURL, dest); err != nil {
		say(T("step.vcredist.fail", err.Error()))
		return nil
	}
	cmd := exec.Command(dest, "/install", "/quiet", "/norestart")
	_ = cmd.Run()
	return nil
}
