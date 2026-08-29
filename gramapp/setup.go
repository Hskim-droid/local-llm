package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func say(s string) { fmt.Println(s) }

func setup(p profile) error {
	say("")
	say("════════════════════════════════════════")
	say("  로컬LLM")
	say("  이 노트북에서만 돌아갑니다.")
	say("  파일은 회사 밖·인터넷으로 안 올라갑니다.")
	say("  Ollama는 필요 없습니다.")
	say("  오류 신고는 " + reportIssuesURL + " 로 주세요.")
	say("════════════════════════════════════════")
	say("")
	say("잠깐 읽어 주세요")
	say("  · 첫 실행은 모델을 받느라 몇 분~십몇 분이 걸립니다.")
	say("  · 그 동안 이 검은 창을 닫지 마세요.")
	say("  · Wi-Fi와 충전기를 연결해 주세요.")
	say("  · Chrome·엣지 탭이 많으면 먼저 닫아 주세요. (메모리)")
	say("  · 원본 파일은 건드리지 않습니다. 옆에 폴더가 생깁니다.")
	say("  · 영상은 옆 .txt가 있으면 그걸 씁니다. 없으면 그때 음성 모델을 받습니다.")
	say("  · PPTX·PDF·DOCX·XLSX·HWPX·HTML·XML·CSV·MD·TXT·영상.")
	say("  · 구형 PPT·XLS·HWP는 각각 PPTX·XLSX·HWPX로 저장한 것만 됩니다.")
	say("  · 스캔·차트는 그때 그림 모델을 받습니다. 채팅 모델과 동시에 안 올립니다.")
	say("")

	gb := ramGB()
	say("[1/3] 이 노트북 사양")
	say(fmt.Sprintf("      RAM  %.0f GB    GPU  %s", gb, gpuName()))
	say(fmt.Sprintf("      → %s", p.Label))
	say(fmt.Sprintf("      받을 모델: %s", strings.Join(p.Pull, "  →  ")))
	if p.ID == "gram16" {
		say("      16GB라서 작은 모델(8B, 약 5GB)만 받습니다.")
		say("      큰 모델(14B)은 이 노트북에서 올리지 않습니다.")
	}
	if p.ID == "gram32" {
		say("      32GB입니다. 8B를 받은 뒤 14B(약 9GB)도 받습니다.")
		say("      14B가 느리면 다음에 8B만 쓰면 됩니다.")
	}
	if av := availGB(); av < 5 {
		say(fmt.Sprintf("      지금 비어 있는 메모리 약 %.1fGB 로 빠듯합니다.", av))
		say("      Chrome·엣지를 닫고 Enter 를 누르세요. 안 닫아도 진행은 됩니다.")
		waitEnter("      Enter > ")
	}
	say("")

	say("[2/3] 엔진 (llama.cpp)")
	if err := ensureLlamaBin(); err != nil {
		return err
	}
	say("      준비됨")
	say("")

	say("[3/3] 모델 받기")
	say("      아래 용량이 올라가면 정상입니다. 창을 닫지 마세요.")
	for i, id := range p.Pull {
		g, ok := specByID(id)
		if !ok {
			continue
		}
		if ggufReady(g) {
			say(fmt.Sprintf("      (%d/%d) %s  이미 있습니다. 건너뜁니다.", i+1, len(p.Pull), g.Label))
			continue
		}
		say(fmt.Sprintf("      (%d/%d) %s", i+1, len(p.Pull), g.Label))
		t0 := time.Now()
		if err := ensureGGUF(g); err != nil {
			if i == 0 {
				return fmt.Errorf("모델 받기 실패 · Wi-Fi를 확인하고 다시 눌러 주세요")
			}
			say("      이 모델은 건너뛰고, 이미 받은 모델로 진행합니다.")
			continue
		}
		say(fmt.Sprintf("      완료 (%s)", time.Since(t0).Round(time.Second)))
	}
	say("")
	say("준비됐습니다.")
	say("  지금은 보고서, 회의록, 번역을 할 수 있습니다.")
	say("")
	return nil
}

func specByID(id string) (ggufSpec, bool) {
	switch id {
	case gguf8b.ID:
		return gguf8b, true
	case gguf14b.ID:
		return gguf14b, true
	default:
		return ggufSpec{}, false
	}
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

var skipPause bool

func waitEnter(prompt string) {
	if skipPause {
		return
	}
	_ = promptLine(prompt)
}
