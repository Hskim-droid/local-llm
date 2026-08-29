package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func pickFiles() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	ps := `
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.OpenFileDialog
$d.Multiselect = $true
$d.Title = '영상, PPT, PDF를 고르세요 (Ctrl로 여러 개)'
$d.Filter = '자료|*.mp4;*.m4a;*.mov;*.wav;*.mp3;*.webm;*.pptx;*.pdf;*.txt;*.md|모든 파일|*.*'
if ($d.ShowDialog() -eq 'OK') { $d.FileNames -join '|' }
`
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", ps)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil
	}
	return strings.Split(s, "|")
}

func reveal(path string) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("explorer", "/select,", path).Start()
		return
	}
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", "-R", path).Start()
	}
}
