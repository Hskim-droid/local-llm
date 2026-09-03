package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	whisperZipWinURL = "https://github.com/ggml-org/whisper.cpp/releases/download/b4938/whisper-bin-x64.zip"
	whisperZipMacURL = "https://github.com/OpenWhispr/whisper.cpp/releases/download/0.0.9/whisper-cpp-darwin-arm64.zip"
	whisperZipMacX64 = "https://github.com/OpenWhispr/whisper.cpp/releases/download/0.0.9/whisper-cpp-darwin-x64.zip"
	whisperModelURL  = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin?download=true"
	ffmpegWinURL     = "https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1/ffmpeg-win32-x64"
	ffmpegMacURL     = "https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1/ffmpeg-darwin-arm64"
	ffmpegMacX64URL  = "https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1/ffmpeg-darwin-x64"
)

type needWhisperError struct{ Path string }

func (e needWhisperError) Error() string {
	return "영상 전사가 필요합니다: " + filepath.Base(e.Path)
}

const (
	toolsVendorASCII     = "LocalLLM"
	toolsVendorHangul    = "로컬LLM"
	toolsVendorHangulOld = "로컬LLM보고서"
)

func toolsDir() string {
	cands := toolsDirCandidatesAt(runtime.GOOS, os.Getenv("LOCALAPPDATA"), homeDir())
	migrateToolsDir(cands)
	cands = toolsDirCandidatesAt(runtime.GOOS, os.Getenv("LOCALAPPDATA"), homeDir())
	for _, p := range cands {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return cands[0]
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func toolsDirCandidatesAt(goos, localApp, home string) []string {
	if goos == "windows" && localApp != "" {
		return []string{
			filepath.Join(localApp, toolsVendorASCII, "tools"),
			filepath.Join(localApp, toolsVendorHangul, "tools"),
			filepath.Join(localApp, toolsVendorHangulOld, "tools"),
		}
	}
	if goos == "darwin" {
		base := filepath.Join(home, "Library", "Application Support")
		return []string{
			filepath.Join(base, toolsVendorASCII, "tools"),
			filepath.Join(base, toolsVendorHangul, "tools"),
			filepath.Join(base, toolsVendorHangulOld, "tools"),
		}
	}
	return []string{filepath.Join(home, ".local-llm", "tools"), filepath.Join(home, ".local-llm-report", "tools")}
}

// whisper-cli on Windows drops Hangul -m paths. Prefer ASCII LocalLLM; move old folders once.
func migrateToolsDir(cands []string) {
	if len(cands) < 2 {
		return
	}
	dest := cands[0]
	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		return
	}
	for _, src := range cands[1:] {
		st, err := os.Stat(src)
		if err != nil || !st.IsDir() {
			continue
		}
		destParent := filepath.Dir(dest)
		srcParent := filepath.Dir(src)
		if err := os.MkdirAll(filepath.Dir(destParent), 0755); err != nil {
			return
		}
		if err := os.Rename(srcParent, destParent); err == nil {
			return
		}
		if err := os.MkdirAll(destParent, 0755); err != nil {
			return
		}
		_ = os.Rename(src, dest)
		return
	}
}

func hasNonASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func asciiArgPath(src string) (string, func(), error) {
	if src == "" || !hasNonASCII(src) {
		return src, func() {}, nil
	}
	dst := filepath.Join(os.TempDir(), "localllm-"+filepath.Base(src))
	if err := copyFile(src, dst); err != nil {
		return src, func() {}, err
	}
	return dst, func() { _ = os.Remove(dst) }, nil
}

func transcribePending(paths []string, chatModel string) {
	if len(paths) == 0 {
		return
	}
	_ = chatModel
	say(T("whisper.need", len(paths)))
	llamaStop()
	if err := ensureWhisper(); err != nil {
		say(T("whisper.tools", err.Error()))
		say(T("whisper.side"))
		return
	}
	for i, p := range paths {
		say(T("whisper.one", i+1, len(paths), filepath.Base(p)))
		if err := transcribeToSidecar(p); err != nil {
			say(T("whisper.fail", err.Error()))
			continue
		}
		say(T("whisper.save", filepath.Base(sidecarTxt(p))))
	}
	say(T("whisper.down"))
}

func ensureWhisper() error {
	if err := ensureFFmpeg(); err != nil {
		return err
	}
	if _, err := whisperCLI(); err != nil {
		url, zipName := whisperZipForOS()
		if url == "" {
			return fmt.Errorf("%s", T("whisper.os"))
		}
		say(T("whisper.get"))
		zipPath := filepath.Join(toolsDir(), zipName)
		if err := downloadFile(url, zipPath); err != nil {
			return err
		}
		dest := filepath.Join(toolsDir(), "whisper")
		if err := unzipTo(zipPath, dest); err != nil {
			return err
		}
	}
	if _, err := whisperCLI(); err != nil {
		return err
	}
	model := whisperModelPath()
	if st, err := os.Stat(model); err != nil || st.Size() < 100<<20 {
		say(T("whisper.model"))
		if err := downloadFile(whisperModelURL, model); err != nil {
			return err
		}
		if st, err := os.Stat(model); err != nil || st.Size() < 100<<20 {
			return fmt.Errorf("%s", T("whisper.short"))
		}
	}
	return nil
}

func ensureFFmpeg() error {
	if p := lookPath("ffmpeg"); p != "" {
		return nil
	}
	local := ffmpegLocalPath()
	if _, err := os.Stat(local); err == nil {
		prependPath(filepath.Dir(local))
		return nil
	}
	url := ffmpegURLForOS()
	if url == "" {
		return fmt.Errorf("%s", T("ffmpeg.miss"))
	}
	say(T("ffmpeg.get"))
	if err := downloadFile(url, local); err != nil {
		return err
	}
	_ = os.Chmod(local, 0755)
	prependPath(filepath.Dir(local))
	return nil
}

func ffmpegLocalPath() string {
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	return filepath.Join(toolsDir(), "ffmpeg", name)
}

func whisperModelPath() string {
	return filepath.Join(toolsDir(), "whisper", "ggml-small.bin")
}

func whisperZipForOS() (url, name string) {
	switch {
	case runtime.GOOS == "windows":
		return whisperZipWinURL, "whisper-bin-x64.zip"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return whisperZipMacURL, "whisper-cpp-darwin-arm64.zip"
	case runtime.GOOS == "darwin":
		return whisperZipMacX64, "whisper-cpp-darwin-x64.zip"
	default:
		return "", ""
	}
}

func ffmpegURLForOS() string {
	switch {
	case runtime.GOOS == "windows":
		return ffmpegWinURL
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return ffmpegMacURL
	case runtime.GOOS == "darwin":
		return ffmpegMacX64URL
	default:
		return ""
	}
}

func isWhisperBin(name string) bool {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".dll"), strings.HasSuffix(n, ".so"), strings.HasSuffix(n, ".dylib"),
		strings.HasSuffix(n, ".txt"), strings.HasSuffix(n, ".bin"), strings.HasSuffix(n, ".zip"):
		return false
	case n == "whisper-cli", n == "whisper-cli.exe", n == "whisper", n == "whisper.exe":
		return true
	case strings.HasPrefix(n, "whisper-cpp"):
		return true
	default:
		return false
	}
}

func whisperCLI() (string, error) {
	for _, name := range []string{"whisper-cli", "whisper"} {
		if p := lookPath(name); p != "" {
			return p, nil
		}
	}
	root := filepath.Join(toolsDir(), "whisper")
	var found string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if isWhisperBin(info.Name()) {
			found = p
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("whisper-cli를 못 찾았습니다")
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(found, 0755)
	}
	prependPath(filepath.Dir(found))
	return found, nil
}

func transcribeToSidecar(media string) error {
	ff := lookPath("ffmpeg")
	if ff == "" {
		ff = ffmpegLocalPath()
	}
	id := fmt.Sprintf("llmr-%d", time.Now().UnixNano())
	wav := filepath.Join(os.TempDir(), id+".wav")
	defer os.Remove(wav)
	cmd := exec.Command(ff, "-nostdin", "-y", "-i", media, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", wav)
	cmd.Stdout = osStdout()
	cmd.Stderr = osStderr()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("영상에서 소리를 못 뽑았습니다")
	}
	cli, err := whisperCLI()
	if err != nil {
		return err
	}
	prefix := filepath.Join(os.TempDir(), id+"-out")
	_ = os.Remove(prefix + ".txt")
	threads := runtime.NumCPU()
	if threads > 4 {
		threads = 4
	}
	model := whisperModelPath()
	modelArg, cleanup, err := asciiArgPath(model)
	if err != nil {
		return err
	}
	defer cleanup()
	wcmd := exec.Command(cli, "-m", modelArg, "-f", wav, "-l", "auto", "-otxt", "-of", prefix, "-t", fmt.Sprintf("%d", threads))
	wcmd.Dir = filepath.Dir(cli)
	wcmd.Stdout = osStdout()
	wcmd.Stderr = osStderr()
	if err := wcmd.Run(); err != nil {
		return fmt.Errorf("전사 실패")
	}
	out := prefix + ".txt"
	b, err := os.ReadFile(out)
	if err != nil {
		return fmt.Errorf("전사 결과를 못 읽었습니다")
	}
	t := strings.TrimSpace(string(b))
	if t == "" {
		return fmt.Errorf("전사문이 비었습니다")
	}
	_ = os.Remove(out)
	return os.WriteFile(sidecarTxt(media), []byte(t+"\n"), 0644)
}

func downloadFile(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	tmp := dest + ".part"
	cli := &http.Client{Timeout: 0}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "local-llm")
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s", T("dl.fail", resp.Status))
	}
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	pw := &progressWriter{total: resp.ContentLength, t0: time.Now()}
	_, err = io.Copy(f, io.TeeReader(resp.Body, pw))
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	say(T("dl.done", filepath.Base(dest)))
	return nil
}

type progressWriter struct {
	total int64
	n     int64
	last  int64
	t0    time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	if p.n-p.last < 30<<20 {
		return len(b), nil
	}
	p.last = p.n
	if p.total > 0 {
		say(T("dl.mb", p.n>>20, p.total>>20))
	} else {
		say(T("dl.mb2", p.n>>20))
	}
	return len(b), nil
}

func unzipTo(zipPath, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if strings.HasPrefix(name, "..") {
			continue
		}
		out := filepath.Join(dest, name)
		rel, err := filepath.Rel(dest, out)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(out, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(out)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, rc)
		w.Close()
		rc.Close()
		if err != nil {
			return err
		}
		if runtime.GOOS != "windows" && isWhisperBin(filepath.Base(out)) {
			_ = os.Chmod(out, 0755)
		}
	}
	return nil
}

func prependPath(dir string) {
	if dir == "" {
		return
	}
	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
