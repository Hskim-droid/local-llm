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
	whisperZipURL   = "https://github.com/ggml-org/whisper.cpp/releases/download/b4938/whisper-bin-x64.zip"
	whisperModelURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin"
	ffmpegWinURL    = "https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1/ffmpeg-win32-x64"
	ffmpegMacURL    = "https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1/ffmpeg-darwin-arm64"
)

type needWhisperError struct{ Path string }

func (e needWhisperError) Error() string {
	return "영상 전사가 필요합니다: " + filepath.Base(e.Path)
}

func toolsDir() string {
	if d := os.Getenv("LOCALAPPDATA"); d != "" && runtime.GOOS == "windows" {
		return filepath.Join(d, "로컬LLM보고서", "tools")
	}
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "로컬LLM보고서", "tools")
	}
	return filepath.Join(home, ".local-llm-report", "tools")
}

func transcribePending(paths []string, chatModel string) {
	if len(paths) == 0 {
		return
	}
	say(fmt.Sprintf("영상 %d개에 전사문이 없습니다. 음성 모델을 씁니다. 보고서 모델은 잠시 내립니다.", len(paths)))
	ollamaStop(chatModel)
	ollamaStop(vlTag)
	if err := ensureWhisper(); err != nil {
		say("  음성 도구 준비 실패: " + err.Error())
		say("  영상 옆에 같은 이름 .txt를 두면 됩니다.")
		return
	}
	for i, p := range paths {
		say(fmt.Sprintf("  전사 %d/%d %s", i+1, len(paths), filepath.Base(p)))
		if err := transcribeToSidecar(p); err != nil {
			say("    실패: " + err.Error())
			continue
		}
		say("    저장 " + filepath.Base(sidecarTxt(p)) + " (원본 영상은 그대로)")
	}
	say("음성 모델을 내렸습니다. 이제 보고서 모델만 씁니다.")
}

func ensureWhisper() error {
	if err := ensureFFmpeg(); err != nil {
		return err
	}
	if _, err := whisperCLI(); err != nil {
		if runtime.GOOS != "windows" {
			return fmt.Errorf("whisper-cli가 없습니다. 맥은 영상 옆 .txt를 두거나 whisper-cli를 PATH에 넣으세요")
		}
		say("  whisper.cpp (약 8MB)를 받습니다. 창을 닫지 마세요.")
		zipPath := filepath.Join(toolsDir(), "whisper-bin-x64.zip")
		if err := downloadFile(whisperZipURL, zipPath); err != nil {
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
		say("  음성 모델 ggml-small (약 470MB)를 받습니다. 창을 닫지 마세요.")
		if err := downloadFile(whisperModelURL, model); err != nil {
			return err
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
	url := ""
	switch {
	case runtime.GOOS == "windows":
		url = ffmpegWinURL
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		url = ffmpegMacURL
	default:
		return fmt.Errorf("ffmpeg가 없습니다. 영상에서 소리를 뽑을 때 필요합니다")
	}
	say("  ffmpeg (영상 소리 뽑기)를 받습니다. 창을 닫지 마세요.")
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

func whisperCLI() (string, error) {
	if p := lookPath("whisper-cli"); p != "" {
		return p, nil
	}
	root := filepath.Join(toolsDir(), "whisper")
	var found string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := strings.ToLower(info.Name())
		if base == "whisper-cli.exe" || base == "whisper-cli" {
			found = p
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("whisper-cli를 못 찾았습니다")
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
	wcmd := exec.Command(cli, "-m", whisperModelPath(), "-f", wav, "-l", "auto", "-otxt", "-of", prefix, "-t", fmt.Sprintf("%d", threads))
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
	req.Header.Set("User-Agent", "local-llm-report")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("다운로드 실패 %s", resp.Status)
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
	say(fmt.Sprintf("      완료 %s", filepath.Base(dest)))
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
		say(fmt.Sprintf("      %d / %d MB", p.n>>20, p.total>>20))
	} else {
		say(fmt.Sprintf("      %d MB", p.n>>20))
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
	}
	return nil
}

func prependPath(dir string) {
	if dir == "" {
		return
	}
	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
