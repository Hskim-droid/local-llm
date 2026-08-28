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
	say("════════════════════════════════════════")
	say("  로컬LLM보고서")
	say("  이 노트북에서만 돌아갑니다.")
	say("  파일은 회사 밖·인터넷으로 안 올라갑니다.")
	say("════════════════════════════════════════")
	say("")
	say("잠깐 읽어 주세요")
	say("  · 첫 실행은 모델을 받느라 몇 분~십몇 분이 걸립니다.")
	say("  · 그 동안 이 검은 창을 닫지 마세요.")
	say("  · Wi-Fi와 충전기를 연결해 주세요.")
	say("  · Chrome·엣지 탭이 많으면 먼저 닫아 주세요. (메모리)")
	say("  · 원본 파일은 건드리지 않습니다. 옆에 폴더가 생깁니다.")
	say("  · 영상은 같은 이름 .txt(전사문)가 옆에 있어야 합니다.")
	say("  · PPT는 PPTX로 저장한 것만 됩니다.")
	say("")

	gb := ramGB()
	say("[1/4] 이 노트북 사양")
	say(fmt.Sprintf("      RAM  %.0f GB    GPU  %s", gb, gpuName()))
	say(fmt.Sprintf("      → %s", p.Label))
	say(fmt.Sprintf("      받을 모델: %s", strings.Join(p.Pull, "  →  ")))
	if p.ID == "gram16" {
		say("      16GB라서 작은 모델(8B, 약 5GB)만 받습니다.")
		say("      큰 모델(14B)은 이 노트북에서 올리지 않습니다.")
	}
	if p.ID == "gram32" {
		say("      32GB입니다. 작은 모델(8B)을 받은 뒤 큰 모델(14B, 약 9GB)도 받습니다.")
		say("      14B가 느리면 다음에 8B만 쓰면 됩니다.")
	}
	if av := availGB(); av < 5 {
		say(fmt.Sprintf("      지금 비어 있는 메모리 약 %.1fGB 로 빠듯합니다.", av))
		say("      Chrome·엣지를 닫고 Enter 를 누르세요. 안 닫아도 진행은 됩니다.")
		waitEnter("      Enter > ")
	}
	say("")

	if lookPath("ollama") == "" {
		say("[2/4] Ollama (로컬 AI) 가 없습니다.")
		say("      문장을 만드는 엔진입니다. 우리 프로그램이 아니라 별도 설치 한 번입니다.")
		say("      설치해도 파일은 이 PC 안에만 있습니다.")
		if runtime.GOOS == "windows" {
			say("      지금 브라우저에서 설치 페이지를 엽니다.")
			say("      1) Windows 용을 받아 설치")
			say("      2) 끝나면 화면 오른쪽 아래 트레이에 라마 아이콘")
			say("      3) 이 창으로 돌아와 Enter")
			_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", "https://ollama.com/download").Start()
			waitEnter("      설치 끝났으면 Enter > ")
		} else {
			return fmt.Errorf("Ollama를 먼저 설치하세요. https://ollama.com")
		}
		if lookPath("ollama") == "" {
			guess := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", "ollama.exe")
			if _, err := os.Stat(guess); err == nil {
				os.Setenv("PATH", filepath.Dir(guess)+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
		}
		if lookPath("ollama") == "" {
			return fmt.Errorf("아직 Ollama를 못 찾았습니다. 설치 후 시작.bat을 다시 눌러 주세요")
		}
	} else {
		say("[2/4] Ollama 있음")
	}
	say("")

	say("[3/4] 엔진 켜기")
	if _, err := ollamaTags(); err != nil {
		say("      서버를 켭니다…")
		ollamaServe()
		if !waitOllama(25) {
			return fmt.Errorf("Ollama를 시작 메뉴에서 한 번 연 다음, 시작.bat을 다시 눌러 주세요")
		}
	}
	say("      켜졌습니다.")
	say("")

	names, _ := ollamaTags()
	say("[4/4] 모델 받기")
	say("      아래 진행 숫자가 올라가면 정상입니다. 창을 닫지 마세요.")
	for i, m := range p.Pull {
		if hasModel(names, m) {
			say(fmt.Sprintf("      (%d/%d) %s  이미 있습니다. 건너뜁니다.", i+1, len(p.Pull), m))
			continue
		}
		sz := "약 5GB, 수 분"
		if strings.Contains(m, "14b") {
			sz = "약 9GB, 십수 분"
		}
		say(fmt.Sprintf("      (%d/%d) %s  (%s)", i+1, len(p.Pull), m, sz))
		t0 := time.Now()
		if err := ollamaPull(m); err != nil {
			if i == 0 {
				return fmt.Errorf("모델 받기 실패: %s  · Wi-Fi를 확인하고 다시 눌러 주세요", m)
			}
			say("      이 모델은 건너뛰고, 이미 받은 모델로 진행합니다.")
			continue
		}
		say(fmt.Sprintf("      완료 (%s)", time.Since(t0).Round(time.Second)))
		names, _ = ollamaTags()
	}
	say("")
	say("준비됐습니다.")
	say("  다음에 파일 고르는 창이 열립니다.")
	say("  Ctrl 을 누른 채 영상·PPT·PDF를 여러 개 고른 뒤 확인을 누르세요.")
	say("  (이 아이콘 위에 파일을 끌어다 놓아도 됩니다.)")
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

func waitEnter(prompt string) {
	fmt.Print(prompt)
	var line string
	_, _ = fmt.Scanln(&line)
}
