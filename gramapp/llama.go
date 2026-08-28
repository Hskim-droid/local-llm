package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	llamaRel       = "b10670"
	llamaZipWin    = "https://github.com/ggml-org/llama.cpp/releases/download/b10670/llama-b10670-bin-win-cpu-x64.zip"
	llamaTarMacARM = "https://github.com/ggml-org/llama.cpp/releases/download/b10670/llama-b10670-bin-macos-arm64.tar.gz"
	llamaTarMacX64 = "https://github.com/ggml-org/llama.cpp/releases/download/b10670/llama-b10670-bin-macos-x64.tar.gz"
	gguf8bURL      = "https://huggingface.co/unsloth/Qwen3-8B-GGUF/resolve/main/Qwen3-8B-Q4_K_M.gguf?download=true"
	gguf14bURL     = "https://huggingface.co/unsloth/Qwen3-14B-GGUF/resolve/main/Qwen3-14B-Q4_K_M.gguf?download=true"
	vlGGUFURL      = "https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF/resolve/main/Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf?download=true"
	vlMMProjURL    = "https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF/resolve/main/mmproj-Qwen2.5-VL-3B-Instruct-Q8_0.gguf?download=true"
	llamaPort      = 18765
)

type ggufSpec struct {
	ID    string
	File  string
	URL   string
	Min   int64
	Label string
}

var (
	gguf8b = ggufSpec{
		ID: "qwen3-8b", File: "Qwen3-8B-Q4_K_M.gguf", URL: gguf8bURL,
		Min: 4 << 30, Label: "Qwen3 8B (약 5GB)",
	}
	gguf14b = ggufSpec{
		ID: "qwen3-14b", File: "Qwen3-14B-Q4_K_M.gguf", URL: gguf14bURL,
		Min: 8 << 30, Label: "Qwen3 14B (약 9GB)",
	}
	vlGGUF = ggufSpec{
		ID: "qwen25vl-3b", File: "Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf", URL: vlGGUFURL,
		Min: 1500 << 20, Label: "그림 모델 3B (약 1.9GB)",
	}
	vlProj = ggufSpec{
		ID: "qwen25vl-mmproj", File: "mmproj-Qwen2.5-VL-3B-Instruct-Q8_0.gguf", URL: vlMMProjURL,
		Min: 700 << 20, Label: "그림 프로젝터 (약 0.8GB)",
	}
)

var (
	llamaMu   sync.Mutex
	llamaCmd  *exec.Cmd
	llamaAddr = fmt.Sprintf("http://127.0.0.1:%d", llamaPort)
)

func modelsDir() string { return filepath.Join(toolsDir(), "models") }
func llamaDir() string  { return filepath.Join(toolsDir(), "llama") }

func ggufPath(g ggufSpec) string { return filepath.Join(modelsDir(), g.File) }

func ggufReady(g ggufSpec) bool {
	st, err := os.Stat(ggufPath(g))
	return err == nil && st.Size() >= g.Min
}

func pickChatGGUF(p profile) (path, label string, err error) {
	specs := chatSpecs(p)
	for i := len(specs) - 1; i >= 0; i-- {
		if ggufReady(specs[i]) {
			return ggufPath(specs[i]), specs[i].Label, nil
		}
	}
	return "", "", fmt.Errorf("모델 파일이 없습니다. 시작.bat을 다시 눌러 받아 주세요")
}

func chatSpecs(p profile) []ggufSpec {
	switch p.ID {
	case "mac24":
		return []ggufSpec{gguf14b}
	case "gram32":
		return []ggufSpec{gguf8b, gguf14b}
	default:
		return []ggufSpec{gguf8b}
	}
}

func ensureLlamaBin() error {
	if _, err := findLlama("llama-server", "llama-server.exe"); err == nil {
		return nil
	}
	url, name := llamaArchiveForOS()
	if url == "" {
		return fmt.Errorf("이 OS용 llama.cpp를 자동으로 못 받습니다")
	}
	say("  llama.cpp 엔진을 받습니다. 창을 닫지 마세요.")
	arch := filepath.Join(toolsDir(), name)
	if err := downloadFile(url, arch); err != nil {
		return err
	}
	dest := llamaDir()
	if strings.HasSuffix(strings.ToLower(name), ".zip") {
		if err := unzipTo(arch, dest); err != nil {
			return err
		}
	} else {
		if err := extractTarGz(arch, dest); err != nil {
			return err
		}
	}
	if _, err := findLlama("llama-server", "llama-server.exe"); err != nil {
		return fmt.Errorf("llama-server를 못 찾았습니다")
	}
	return nil
}

func llamaArchiveForOS() (url, name string) {
	switch {
	case runtime.GOOS == "windows":
		return llamaZipWin, "llama-" + llamaRel + "-bin-win-cpu-x64.zip"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return llamaTarMacARM, "llama-" + llamaRel + "-bin-macos-arm64.tar.gz"
	case runtime.GOOS == "darwin":
		return llamaTarMacX64, "llama-" + llamaRel + "-bin-macos-x64.tar.gz"
	default:
		return "", ""
	}
}

func findLlama(names ...string) (string, error) {
	want := map[string]bool{}
	for _, n := range names {
		want[strings.ToLower(n)] = true
	}
	var found string
	_ = filepath.Walk(llamaDir(), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if want[strings.ToLower(info.Name())] {
			found = p
			return io.EOF
		}
		return nil
	})
	if found == "" {
		if p := lookPath(names[0]); p != "" {
			return p, nil
		}
		return "", fmt.Errorf("%s 없음", names[0])
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(found, 0755)
	}
	prependPath(filepath.Dir(found))
	return found, nil
}

func ensureGGUF(g ggufSpec) error {
	if ggufReady(g) {
		return nil
	}
	say("  " + g.Label + " 를 받습니다. 창을 닫지 마세요.")
	if err := downloadFile(g.URL, ggufPath(g)); err != nil {
		return err
	}
	if !ggufReady(g) {
		return fmt.Errorf("%s 가 덜 받았습니다. 다시 눌러 주세요", g.File)
	}
	return nil
}

func llamaStart(gguf string, p profile) error {
	llamaStop()
	bin, err := findLlama("llama-server", "llama-server.exe")
	if err != nil {
		return err
	}
	ngl := 0
	if runtime.GOOS == "darwin" {
		ngl = 99
	}
	th := runtime.NumCPU()
	if p.ID == "gram16" && th > 4 {
		th = 4
	}
	if th > 8 {
		th = 8
	}
	args := []string{
		"-m", gguf,
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", llamaPort),
		"-c", fmt.Sprintf("%d", p.Ctx),
		"-ngl", fmt.Sprintf("%d", ngl),
		"-t", fmt.Sprintf("%d", th),
		"--jinja",
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Dir(bin)
	cmd.Stdout = osStdout()
	cmd.Stderr = osStderr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("엔진을 못 켰습니다: %v", err)
	}
	llamaMu.Lock()
	llamaCmd = cmd
	llamaMu.Unlock()
	say("  모델을 올리는 중… (이 창을 닫지 마세요)")
	if !waitLlama(90) {
		llamaStop()
		return fmt.Errorf("모델이 안 켜졌습니다. Chrome을 닫고 다시 눌러 주세요")
	}
	return nil
}

func waitLlama(sec int) bool {
	cli := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(time.Duration(sec) * time.Second)
	for time.Now().Before(deadline) {
		llamaMu.Lock()
		cmd := llamaCmd
		llamaMu.Unlock()
		if cmd == nil || cmd.Process == nil {
			return false
		}
		resp, err := cli.Get(llamaAddr + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	return false
}

func llamaStop() {
	llamaMu.Lock()
	cmd := llamaCmd
	llamaCmd = nil
	llamaMu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func llamaChatJSON(model, system, user string, ctx, predict int) (map[string]any, error) {
	_ = ctx
	if strings.Contains(strings.ToLower(model), "qwen3") && !strings.Contains(user, "/no_think") {
		user = "/no_think\n" + user
	}
	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.1,
		"max_tokens":  predict,
		"stream":      false,
	})
	cli := &http.Client{Timeout: 240 * time.Second}
	resp, err := cli.Post(llamaAddr+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil || len(wrap.Choices) == 0 {
		return parseJSONObj(string(raw))
	}
	return parseJSONObj(wrap.Choices[0].Message.Content)
}

func parseJSONObj(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "</think>"); i >= 0 {
		s = strings.TrimSpace(s[i+8:])
	}
	a, b := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if a < 0 || b <= a {
		return nil, fmt.Errorf("JSON 없음")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s[a:b+1]), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func extractTarGz(src, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		out := filepath.Join(dest, name)
		rel, err := filepath.Rel(dest, out)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if hdr.Typeflag == tar.TypeDir {
			_ = os.MkdirAll(out, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		w, err := os.Create(out)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, tr)
		w.Close()
		if err != nil {
			return err
		}
		if runtime.GOOS != "windows" {
			_ = os.Chmod(out, os.FileMode(hdr.Mode)|0755)
		}
	}
	return nil
}

func llamaChatVision(system, user string, img []byte, ext string) (map[string]any, error) {
	bin, err := findLlama("llama-mtmd-cli", "llama-mtmd-cli.exe")
	if err != nil {
		return nil, err
	}
	if !ggufReady(vlGGUF) || !ggufReady(vlProj) {
		return nil, fmt.Errorf("그림 모델 파일 없음")
	}
	if ext == "" {
		ext = ".jpg"
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("llmr-vl-%d%s", time.Now().UnixNano(), ext))
	if err := os.WriteFile(tmp, img, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	prompt := system + "\n" + user
	cmd := exec.Command(bin,
		"-m", ggufPath(vlGGUF),
		"--mmproj", ggufPath(vlProj),
		"--image", tmp,
		"-p", prompt,
		"--temp", "0.1",
		"-n", "400",
	)
	cmd.Dir = filepath.Dir(bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("그림 모델 실패")
	}
	return parseJSONObj(string(out))
}
