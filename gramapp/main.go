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
	harvestOnly := false
	force8b := false
	var rawFiles []string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch {
		case a == "--harvest":
			harvestOnly = true
		case a == "--8b":
			force8b = true
		case a == "--yes":
			skipPause = true
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

	var files []string
	for _, f := range rawFiles {
		if abs, err := filepath.Abs(f); err == nil {
			if st, err := os.Stat(abs); err == nil && st.Mode().IsRegular() {
				files = append(files, abs)
			}
		}
	}

	p := pickProfile(ramGB())
	if force8b {
		p.Pull = []string{"qwen3-8b"}
	}
	if harvestOnly {
		if len(files) == 0 {
			say("--harvest 뒤에 파일 경로를 붙여 주세요. 모델은 안 받습니다.")
			os.Exit(2)
		}
		out, err := runHarvestOnly(files)
		if err != nil {
			say("실패: " + err.Error())
			reportFailure(p, "", files, err)
			os.Exit(1)
		}
		say("수확만 했습니다. 모델은 안 올렸습니다.")
		say("  " + out)
		return
	}

	if err := setup(p); err != nil {
		say("오류: " + err.Error())
		reportFailure(p, "", files, err)
		pause()
		os.Exit(1)
	}

	model, label, err := pickChatGGUF(p)
	if err != nil {
		say("오류: " + err.Error())
		reportFailure(p, "", files, err)
		pause()
		os.Exit(1)
	}
	say("지금 쓰는 모델: " + label)
	if availGB() < 4 {
		say("메모리가 빠듯합니다. Chrome·엣지를 닫으면 속도와 워드 저장이 안정됩니다.")
	}

	if packArg != "" {
		pk, err := pickPack(packArg)
		if err != nil {
			say("오류: " + err.Error())
			reportFailure(p, packArg, files, err)
			pause()
			os.Exit(1)
		}
		if err := runOneJob(files, p, model, pk, true); err != nil {
			os.Exit(1)
		}
		pause()
		return
	}

	runLoop(files, p, model)
}

func runLoop(pending []string, p profile, model string) {
	for {
		packs, err := catalogPacks()
		if err != nil {
			say("오류: " + err.Error())
			reportFailure(p, "", pending, err)
			pause()
			return
		}
		sayCanDo(packs)
		line := strings.TrimSpace(promptLine("> "))
		if isQuit(line) || (skipPause && line == "") {
			say("종료합니다.")
			return
		}
		if line == "" {
			continue
		}
		pk, ok := matchPack(packs, line)
		if !ok {
			say("지금은 목록에 있는 것만 할 수 있습니다.")
			continue
		}
		pk, err = ensurePackLoaded(pk)
		if err != nil {
			say("오류: " + err.Error())
			reportFailure(p, pk.ID, pending, err)
			continue
		}
		files := pending
		pending = nil
		if err := runOneJob(files, p, model, pk, false); err != nil {
			continue
		}
	}
}

func runOneJob(files []string, p profile, model string, pk pack, failHard bool) error {
	say("용도: " + pk.Label)
	if len(files) == 0 {
		files = pickFiles()
	}
	if len(files) == 0 {
		raw := strings.TrimSpace(promptLine("파일 경로 > "))
		if raw != "" {
			if abs, err := filepath.Abs(raw); err == nil {
				if st, err := os.Stat(abs); err == nil && st.Mode().IsRegular() {
					files = []string{abs}
				}
			}
		}
	}
	if len(files) == 0 {
		say("파일을 고르지 않았습니다.")
		if failHard {
			say("  · 시작.bat을 다시 눌러 창에서 고르거나")
			say("  · 로컬LLM 아이콘 위에 파일을 끌어다 놓으세요.")
			pause()
		}
		return fmt.Errorf("no files")
	}
	t0 := time.Now()
	out, err := runFiles(files, p, model, pk)
	llamaStop()
	if err != nil {
		say("실패: " + err.Error())
		say("원본 파일은 그대로 있습니다.")
		reportFailure(p, pk.ID, files, err)
		if failHard {
			pause()
		}
		return err
	}
	say("")
	say("끝났습니다. 원본은 그대로 두었습니다.")
	say("  " + out)
	say(fmt.Sprintf("걸린 시간 %s · 모델은 내려서 메모리를 비웠습니다.", time.Since(t0).Round(time.Second)))
	reveal(out)
	return nil
}

func pause() {
	if skipPause {
		return
	}
	_ = promptLine("\n창을 닫으려면 Enter > ")
}

func mkdirAll(p string) error {
	return os.MkdirAll(p, 0755)
}
