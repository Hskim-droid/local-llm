package main

import (
	"fmt"
	"os/exec"
	"time"
)

func say(s string) { fmt.Println(s) }

func setup(p profile) error {
	first := !chatModelReady(p)
	say("")
	say("════════════════════════════════════════")
	say("  로컬LLM  " + appVersion)
	say("  이 노트북에서만 돌아갑니다. 파일은 나가지 않습니다.")
	say("  오류 신고는 " + reportIssuesURL + " 로 주세요.")
	say("════════════════════════════════════════")
	if first {
		say("")
		say("잠깐 읽어 주세요 (첫 실행만)")
		say("  · 모델을 받느라 몇 분~십몇 분이 걸립니다. 이 창을 닫지 마세요.")
		say("  · Wi-Fi와 충전기를 연결해 주세요. Chrome 탭은 줄이세요.")
		say("  · 원본은 그대로입니다. 결과는 옆에 폴더로 생깁니다.")
		say("  · PPTX·PDF·DOCX·XLSX·HWPX·HTML·XML·CSV·MD·TXT·영상.")
		say("  · 구형 PPT·XLS·HWP는 PPTX·XLSX·HWPX로 저장하세요.")
	}
	say("")

	if first {
		gb := ramGB()
		say("[1/3] 이 노트북 사양")
		say(fmt.Sprintf("      RAM  %.0f GB    GPU  %s", gb, gpuName()))
		say(fmt.Sprintf("      → %s", p.Label))
		if p.ID == "gram16" {
			say("      16GB라서 작은 모델(8B, 약 5GB)만 받습니다.")
		}
		if p.ID == "gram32" {
			say("      32GB입니다. 8B 다음 14B(약 9GB)도 받습니다.")
		}
		if av := availGB(); av < 5 {
			say(fmt.Sprintf("      지금 비어 있는 메모리 약 %.1fGB 로 빠듯합니다.", av))
			say("      Chrome·엣지를 닫고 Enter 를 누르세요. 안 닫아도 진행은 됩니다.")
			waitEnter("      Enter > ")
		}
		say("")
		say("[2/3] 엔진 (llama.cpp)")
	}
	if err := ensureLlamaBin(); err != nil {
		return err
	}
	if first {
		say("      준비됨")
		say("")
		say("[3/3] 모델 받기")
		say("      아래 용량이 올라가면 정상입니다. 창을 닫지 마세요.")
	}
	for i, id := range p.Pull {
		g, ok := specByID(id)
		if !ok {
			continue
		}
		if ggufReady(g) {
			if first {
				say(fmt.Sprintf("      (%d/%d) %s  이미 있습니다. 건너뜁니다.", i+1, len(p.Pull), g.Label))
			}
			continue
		}
		if !first {
			say(fmt.Sprintf("모델 받는 중: %s  창을 닫지 마세요.", g.Label))
		} else {
			say(fmt.Sprintf("      (%d/%d) %s", i+1, len(p.Pull), g.Label))
		}
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
	if first {
		say("")
		say("준비됐습니다.")
		say("")
	}
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
