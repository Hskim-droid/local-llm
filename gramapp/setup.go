package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func say(s string) { fmt.Println(s) }

func setup(p profile) error {
	say("")
	say("════════════════════════════════════")
	say("  로컬LLM보고서  ·  더블클릭")
	say("  영상·PPT·PDF → 한글 워드")
	say("════════════════════════════════════")
	say("")
	gb := ramGB()
	say("[1] 이 노트북")
	say(fmt.Sprintf("    RAM  %.0f GB   GPU  %s", gb, gpuName()))
	say(fmt.Sprintf("    → %s  · 받을 모델: %s", p.Label, strings.Join(p.Pull, " → ")))
	if p.ID == "gram16" {
		say("    16GB라 8B만 받습니다. 14B는 올리지 않습니다.")
	}
	if av := availGB(); av < 4 {
		say(fmt.Sprintf("    경고: 가용 메모리 약 %.1fGB. Chrome·엣지를 닫아 주세요.", av))
	}
	say("")

	if lookPath("ollama") == "" {
		say("[2] Ollama가 없습니다. 로컬 AI 엔진입니다. 문서는 PC 밖으로 안 나갑니다.")
		if runtime.GOOS == "windows" {
			say("    브라우저가 열리면 Windows용을 설치하세요.")
			say("    설치가 끝나면 트레이에 라마 아이콘이 뜹니다. 이 창으로 돌아와 Enter.")
			_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", "https://ollama.com/download").Start()
			waitEnter()
		} else {
			return fmt.Errorf("Ollama를 먼저 설치하세요: https://ollama.com")
		}
		if lookPath("ollama") == "" {
			// installer may not refresh PATH in this process
			guess := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", "ollama.exe")
			if _, err := os.Stat(guess); err == nil {
				os.Setenv("PATH", filepath.Dir(guess)+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
		}
	} else {
		say("[2] Ollama 있음")
	}
	say("")

	say("[3] 엔진 켜기")
	if _, err := ollamaTags(); err != nil {
		ollamaServe()
		if !waitOllama(25) {
			return fmt.Errorf("Ollama를 시작 메뉴에서 한 번 실행한 뒤 다시 눌러 주세요")
		}
	}
	say("    준비됨")
	say("")

	names, _ := ollamaTags()
	say("[4] 모델 받기  (Wi-Fi, 충전기 연결)")
	for i, m := range p.Pull {
		if hasModel(names, m) {
			say(fmt.Sprintf("    (%d/%d) %s  이미 있음", i+1, len(p.Pull), m))
			continue
		}
		sz := "약 5GB"
		if strings.Contains(m, "14b") {
			sz = "약 9GB"
		}
		say(fmt.Sprintf("    (%d/%d) %s  %s  받는 중… 진행률은 아래", i+1, len(p.Pull), m, sz))
		t0 := time.Now()
		if err := ollamaPull(m); err != nil {
			if i == 0 {
				return fmt.Errorf("모델 받기 실패: %s", m)
			}
			say("    이 모델은 건너뜁니다.")
			continue
		}
		say(fmt.Sprintf("    완료 (%s)", time.Since(t0).Round(time.Second)))
		names, _ = ollamaTags()
	}
	say("")
	say("준비됐습니다. 파일 선택 창이 열립니다. Ctrl로 여러 개를 고르세요.")
	say("")
	return nil
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func waitEnter() {
	fmt.Print("    설치 끝났으면 Enter > ")
	var line string
	_, _ = fmt.Scanln(&line)
}
