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

	p := pickProfile(ramGB())
	if err := setup(p); err != nil {
		say("오류: " + err.Error())
		pause()
		os.Exit(1)
	}

	files := os.Args[1:]
	var cleaned []string
	for _, f := range files {
		if strings.HasPrefix(f, "-") {
			continue
		}
		if abs, err := filepath.Abs(f); err == nil {
			if st, err := os.Stat(abs); err == nil && st.Mode().IsRegular() {
				cleaned = append(cleaned, abs)
			}
		}
	}
	files = cleaned
	if len(files) == 0 {
		files = pickFiles()
	}
	if len(files) == 0 {
		say("파일이 없습니다. 이 프로그램 아이콘에 파일을 끌어다 놓거나, 창에서 고르세요.")
		pause()
		os.Exit(2)
	}

	names, _ := ollamaTags()
	model := pickInstalled(p.Pull, names)
	if model == "" {
		say("모델이 없습니다. 창을 닫지 말고 다시 실행해 주세요.")
		pause()
		os.Exit(1)
	}
	say("모델 " + model)
	if availGB() < 4 {
		say("메모리가 빠듯합니다. Chrome·엣지를 닫으면 더 잘 돌아갑니다.")
	}

	t0 := time.Now()
	out, err := runFiles(files, p, model)
	ollamaStop(model)
	if err != nil {
		say("실패: " + err.Error())
		pause()
		os.Exit(1)
	}
	say("")
	say("저장 위치")
	say("  " + out)
	say(fmt.Sprintf("완료 %s · 모델을 내려 메모리를 비웠습니다.", time.Since(t0).Round(time.Second)))
	reveal(out)
	pause()
}

func pause() {
	fmt.Print("\n창을 닫으려면 Enter > ")
	var s string
	_, _ = fmt.Scanln(&s)
}

func mkdirAll(p string) error {
	return os.MkdirAll(p, 0755)
}
