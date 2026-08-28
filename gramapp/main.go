package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	setUTF8Console()
	os.Setenv("PYTHONUTF8", "1")

	packArg := ""
	var rawFiles []string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch {
		case a == "--pack" && i+1 < len(os.Args):
			packArg = os.Args[i+1]
			i++
		case strings.HasPrefix(a, "--pack="):
			packArg = strings.TrimPrefix(a, "--pack=")
		case strings.HasPrefix(a, "-"):
			continue
		default:
			rawFiles = append(rawFiles, a)
		}
	}

	p := pickProfile(ramGB())
	if err := setup(p); err != nil {
		say("오류: " + err.Error())
		pause()
		os.Exit(1)
	}

	pk, err := pickPack(packArg)
	if err != nil {
		say("오류: " + err.Error())
		pause()
		os.Exit(1)
	}
	say("용도: " + pk.Label)

	var files []string
	for _, f := range rawFiles {
		if abs, err := filepath.Abs(f); err == nil {
			if st, err := os.Stat(abs); err == nil && st.Mode().IsRegular() {
				files = append(files, abs)
			}
		}
	}
	if len(files) == 0 {
		files = pickFiles()
	}
	if len(files) == 0 {
		say("파일을 고르지 않았습니다.")
		say("  · 시작.bat을 다시 눌러 창에서 고르거나")
		say("  · 로컬LLM보고서 아이콘 위에 파일을 끌어다 놓으세요.")
		pause()
		os.Exit(2)
	}

	model, label, err := pickChatGGUF(p)
	if err != nil {
		say("오류: " + err.Error())
		pause()
		os.Exit(1)
	}
	say("지금 쓰는 모델: " + label)
	if availGB() < 4 {
		say("메모리가 빠듯합니다. Chrome·엣지를 닫으면 속도와 워드 저장이 안정됩니다.")
	}

	t0 := time.Now()
	out, err := runFiles(files, p, model, pk)
	llamaStop()
	if err != nil {
		say("실패: " + err.Error())
		say("원본 파일은 그대로 있습니다.")
		pause()
		os.Exit(1)
	}
	say("")
	say("끝났습니다. 원본은 그대로 두었습니다.")
	say("저장 위치 (이 폴더가 곧 열립니다)")
	say("  " + out)
	say(fmt.Sprintf("걸린 시간 %s · 모델은 내려서 메모리를 비웠습니다.", time.Since(t0).Round(time.Second)))
	reveal(out)
	pause()
}

func pause() {
	_ = promptLine("\n창을 닫으려면 Enter > ")
}

func mkdirAll(p string) error {
	return os.MkdirAll(p, 0755)
}
