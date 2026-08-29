package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var uiLang = "en"

func T(key string, args ...any) string {
	s, ok := catalogs[uiLang][key]
	if !ok {
		s = catalogs["en"][key]
	}
	if s == "" {
		s = key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

func normalizeLang(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	switch {
	case s == "ko" || strings.HasPrefix(s, "ko-") || s == "kr" || s == "kor" || s == "korean":
		return "ko"
	case s == "en" || strings.HasPrefix(s, "en-") || s == "eng" || s == "english":
		return "en"
	default:
		return ""
	}
}

func langFile() string { return filepath.Join(appDir(), "lang.txt") }

func loadSavedLang() string {
	b, err := os.ReadFile(langFile())
	if err != nil {
		return ""
	}
	return normalizeLang(string(b))
}

func saveLang(lang string) {
	lang = normalizeLang(lang)
	if lang == "" {
		return
	}
	_ = os.MkdirAll(appDir(), 0755)
	_ = os.WriteFile(langFile(), []byte(lang+"\n"), 0644)
}

func resolveLang(flag string) string {
	if l := normalizeLang(flag); l != "" {
		return l
	}
	if l := normalizeLang(os.Getenv("LOCAL_LLM_LANG")); l != "" {
		return l
	}
	if l := loadSavedLang(); l != "" {
		return l
	}
	return osUILang()
}

func setUILang(lang string) {
	if lang != "ko" {
		lang = "en"
	}
	uiLang = lang
}

func packAlias(id string) []string {
	switch id {
	case "보고서":
		return []string{"보고서", "report", "reports"}
	case "회의록":
		return []string{"회의록", "minutes", "minute"}
	case "번역":
		return []string{"번역", "translate", "translation"}
	default:
		return []string{id}
	}
}

func packLabel(p pack) string {
	if uiLang == "ko" {
		return p.Label
	}
	switch p.ID {
	case "보고서":
		return "Report"
	case "회의록":
		return "Minutes"
	case "번역":
		return "Translation"
	default:
		return p.Label
	}
}

func packOutputs(p pack) (suffix, name, title, body string) {
	if uiLang == "ko" {
		return p.OutSuffix, p.OutName, p.TitleSuffix, p.BodyHeading
	}
	switch p.ID {
	case "보고서":
		return "_report", "report.docx", "report", "Contents"
	case "회의록":
		return "_minutes", "minutes.docx", "minutes", "Minutes"
	case "번역":
		return "_translation", "translation.docx", "translation", "Translation"
	default:
		return p.OutSuffix, p.OutName, p.TitleSuffix, p.BodyHeading
	}
}

func withLang(sys string) string {
	return T("llm.lang") + "\n" + sys
}

func packBlurb(p pack) string {
	if uiLang == "ko" {
		return p.Blurb
	}
	switch p.ID {
	case "보고서":
		return "Slides, sheets, Word, HWPX, web, and video into a report"
	case "회의록":
		return "Transcripts, slides, sheets, and web into minutes"
	case "번역":
		return "Into the UI language by default; say the target language to change it"
	default:
		if p.Blurb != "" {
			return p.Blurb
		}
		return p.Label
	}
}

func profileLabel(p profile) string {
	if uiLang == "ko" {
		return p.Label
	}
	switch p.ID {
	case "mac24":
		return "Mac 24GB"
	case "gram32":
		return "gram 32GB"
	default:
		return "gram 16GB"
	}
}

func modelLabel(g ggufSpec) string {
	key := "model." + g.ID
	if catalogs[uiLang][key] != "" || catalogs["en"][key] != "" {
		return T(key)
	}
	return g.Label
}

var catalogs = map[string]map[string]string{
	"en": {
		"banner.local":          "Runs only on this PC. Files do not leave the machine.",
		"banner.report":         "Send error reports to %s",
		"first.title":           "Read this once (first run only)",
		"first.model":           "  · Downloading the model takes several minutes. Do not close this window.",
		"first.wifi":            "  · Stay on Wi-Fi and power. Close extra Chrome tabs.",
		"first.orig":            "  · Originals are left untouched. Output goes in a new folder beside them.",
		"first.types":           "  · PPTX, PDF, DOCX, XLSX, HWPX, HTML, XML, CSV, MD, TXT, video.",
		"first.old":             "  · Old PPT / XLS / HWP: save as PPTX / XLSX / HWPX first.",
		"step.hw":               "[1/3] This PC",
		"step.hw.ram":           "      RAM  %.0f GB    GPU  %s",
		"step.hw.profile":       "      → %s",
		"step.hw.16":            "      16GB: 8B only (~5GB).",
		"step.hw.32":            "      32GB: 8B, then 14B (~9GB).",
		"step.hw.low":           "      Only about %.1fGB free right now.",
		"step.hw.low2":          "      Close Chrome/Edge and press Enter. You can continue without closing.",
		"step.engine":           "[2/3] Engine (llama.cpp)",
		"step.ready":            "      Ready",
		"step.models":           "[3/3] Models",
		"step.models.hint":      "      Growing file size is normal. Do not close this window.",
		"step.have":             "      (%d/%d) %s  already there. Skipping.",
		"step.get":              "      (%d/%d) %s",
		"step.get2":             "Downloading %s. Do not close this window.",
		"step.skip":             "      Skipping this model; continuing with what is already here.",
		"step.done":             "      Done (%s)",
		"step.prepared":         "Ready.",
		"err.model.dl":          "Model download failed. Check Wi-Fi and run again.",
		"cando.now":             "Right now you can do %s.",
		"cando.quit":            "  Type quit to exit.",
		"cando.only":            "Only items on this list are available right now.",
		"prompt.quit":           "\nPress Enter to close > ",
		"prompt.file":           "File path > ",
		"prompt.enter":          "      Enter > ",
		"using.model":           "Model: %s",
		"mem.low":               "Memory is tight. Closing Chrome/Edge helps speed and Word save.",
		"job.pack":              "Job: %s",
		"job.nofile":            "No file selected.",
		"job.nofile.bat":        "  · Run 시작.bat again and pick files in the window, or",
		"job.nofile.drop":       "  · Drop files onto the local-llm icon.",
		"job.fail":              "Failed: %s",
		"job.orig":              "Original files are unchanged.",
		"job.done":              "Done. Originals were left as they were.",
		"job.time":              "Took %s. The model was unloaded.",
		"bye":                   "Bye.",
		"err.prefix":            "Error: %s",
		"harvest.need":          "Put file paths after --harvest. No model is downloaded.",
		"harvest.ok":            "Harvest only. No model was loaded.",
		"harvest.skip.vid":      "  skip %s — video harvest needs a sidecar .txt",
		"harvest.skip":          "  skip %s — %s",
		"harvest.empty":         "Nothing to extract",
		"harvest.stats":         "segments %d · harvested numbers %d",
		"run.skip.kind":         "  skip %s — not a type this engine reads",
		"run.skip":              "  skip %s — %s",
		"run.empty":             "No text extracted (a scan PDF may need the vision model)",
		"run.extract":           "extract %d files · %d segments · pack %s",
		"run.harvest":           "harvested %d numbers (code). Extra numbers will be stripped.",
		"run.merge":             "merge shared %d · unique %d · conflict %d",
		"run.fold":              "Source is large; folding before assemble.",
		"run.verify.n":          "verify: replaced %d numbers not in the source with [check source].",
		"run.verify.ok":         "verify: no extra numbers",
		"report.send":           "Send error reports to %s",
		"report.only":           "Do not send source files. Attach error.txt only.",
		"note.title":            "local-llm error note",
		"note.body":             "Attach this file to an issue. Do not attach source, Word, or extracted text.",
		"note.send":             "Send error reports to %s",
		"note.time":             "time: %s",
		"note.ver":              "version: %s",
		"note.os":               "OS: %s/%s",
		"note.profile":          "profile: %s (%s)",
		"note.ram":              "RAM: %.0f GB (free %.1f GB)",
		"note.pack":             "pack: %s",
		"note.pack.none":        "pack: (none)",
		"note.ext":              "input extensions: %s",
		"note.ext.none":         "input extensions: (none)",
		"note.err":              "error: %s",
		"err.pack.none":         "No form packs found",
		"err.pack.unk":          "Unknown job: %s",
		"err.quit":              "quit",
		"err.pack.fetch":        "That pack is not on the allowlist",
		"err.pack.get":          "Could not fetch pack: %v",
		"err.pack.read":         "Could not read fetched pack: %s",
		"pack.fetch":            "  Fetching pack from the public repo: %s",
		"err.old.ppt":           "Old .ppt is not supported. Save as PPTX.",
		"err.old.xls":           "Old .xls is not supported. Save as XLSX.",
		"err.old.hwp":           "Old .hwp is not supported. Save as HWPX.",
		"err.kind":              "Unsupported type: %s",
		"err.zip.text":          "No text in %s",
		"err.no.model":          "No model file. Run 시작.bat again to download it.",
		"err.llama.os":          "No llama.cpp build for this OS",
		"err.llama.get":         "  Downloading llama.cpp. Do not close this window.",
		"err.llama.bin":         "llama-server not found",
		"err.gguf.short":        "%s is incomplete. Run again.",
		"err.engine":            "Could not start the engine: %v",
		"err.model.up":          "Model did not start. Close Chrome and try again.",
		"model.up":              "  Loading the model… (do not close this window)",
		"model.get":             "  Downloading %s. Do not close this window.",
		"model.qwen3-8b":        "Qwen3 8B (~5GB)",
		"model.qwen3-14b":       "Qwen3 14B (~9GB)",
		"model.qwen25vl-3b":     "Vision 3B (~1.9GB)",
		"model.qwen25vl-mmproj": "Vision projector (~0.8GB)",
		"vision.thin":           "%d thin chart/scan image(s). Unloading chat, loading vision. Not together.",
		"vision.miss":           "  No vision model: %s  — continuing with text only.",
		"vision.one":            "  image %d/%d (%s)",
		"vision.skip":           "    skip: %s",
		"vision.down":           "Vision model unloaded. Chat model only now.",
		"vision.dl":             "  Vision model for scans/charts. Not loaded with the chat model.",
		"whisper.need":          "%d video(s) have no transcript. Using the speech model. Chat model is unloaded for now.",
		"whisper.tools":         "  Speech tools failed: %s",
		"whisper.side":          "  Put a .txt with the same name next to the video.",
		"whisper.one":           "  transcribe %d/%d %s",
		"whisper.fail":          "    failed: %s",
		"whisper.save":          "    saved %s (video unchanged)",
		"whisper.down":          "Speech model unloaded. Chat model only now.",
		"whisper.os":            "No whisper.cpp for this OS. Put a same-name .txt next to the video.",
		"whisper.get":           "  Downloading whisper.cpp. Do not close this window.",
		"whisper.model":         "  Downloading speech model ggml-small (~470MB). Do not close this window.",
		"whisper.short":         "Speech model is incomplete. Run again.",
		"ffmpeg.miss":           "ffmpeg is missing (needed to pull audio from video)",
		"ffmpeg.get":            "  Downloading ffmpeg. Do not close this window.",
		"dl.fail":               "download failed %s",
		"dl.done":               "      done %s",
		"dl.mb":                 "      %d / %d MB",
		"dl.mb2":                "      %d MB",
		"dialog.title":          "Choose files (Ctrl for multiple)",
		"dialog.filter":         "Files",
		"dialog.all":            "All files",
		"llm.lang":              "OUTPUT LANGUAGE: English. Every JSON title, heading, fact, and bullet must be English. Keep proper names and numbers as in the source. This overrides pack prompts that ask for another language.",
		"llm.chunk":             "Location:%s\n%s\nTurn this source into fact JSON.\n{\"heading_hint\":\"\",\"facts\":[\"noun phrases\"],\"rows\":[]}\n\n%s",
		"llm.assemble":          "Title candidate:%s\n%s\nFacts only, grouped JSON.\n{\"title\":\"\",\"date\":\"\",\"time\":\"\",\"place\":\"\",\"attendees\":\"\",\"agenda\":\"\",\"exec\":[\"\"],\"sections\":[{\"heading\":\"1. Overview\",\"facts\":[\"\"]}],\"actions\":[\"\"]}\n\n%s",
		"llm.fold":              "You compress fact lists. Deduplicate. Keep numbers as written. No invention. JSON only. {\"facts\":[\"noun phrases\"]}",
		"llm.fold.user":         "%s\nDeduplicate this fact list to JSON.\n{\"facts\":[\"noun phrases\"]}\n\n%s",
		"llm.vision":            "You read scans and charts. Visible text and numbers only, in the output language. No invention. JSON only. {\"facts\":[\"noun phrases\"],\"numbers\":[\"as written\"]}",
		"llm.vision.user":       "Visible numbers and text as JSON.",
		"harvest.allow":         "Source numbers (do not invent numbers not on this list; keep original spelling):\n",
		"merge.intro":           "Facts grouped by code. Shared = same in several sources. Unique = one source. Conflict = keep both; do not pick a side.\n",
		"merge.side.unique":     "Unique facts go in an engine section; do not repeat them in the body. Conflicts go in an engine section too.\n\n",
		"merge.side.body":       "Put unique facts in the body. Conflicts go in an engine section.\n\n",
		"merge.shared":          "[shared]",
		"merge.unique":          "[unique]",
		"merge.unique.skip":     "[unique] Do not repeat in the body. The engine will append them.\n\n",
		"merge.conflict":        "[conflict] Do not pick a side. The engine will append them.\n",
		"out.check":             "[check source]",
		"out.missing":           "(not in source)",
		"out.date":              "Date",
		"out.time":              "Time",
		"out.place":             "Place",
		"out.people":            "Attendees",
		"out.agenda":            "Agenda",
		"out.exec":              "0. Executive Summary",
		"out.next":              "Next steps",
		"out.unique":            "Only in one source",
		"out.conflict":          "Conflicting sources",
		"out.fallback.exec":     "Source notes",
		"out.fallback.act":      "Needs follow-up",
		"out.error.file":        "error.txt",
		"out.harvest.dir":       "_harvest",
		"out.harvest.txt":       "extract.txt",
		"out.json.harvest":      "harvest.json",
		"out.json.verify":       "verify.json",
		"out.json.merge":        "merge.json",
	},
	"ko": {
		"banner.local":          "이 노트북에서만 돌아갑니다. 파일은 나가지 않습니다.",
		"banner.report":         "오류 신고는 %s 로 주세요.",
		"first.title":           "잠깐 읽어 주세요 (첫 실행만)",
		"first.model":           "  · 모델을 받느라 몇 분~십몇 분이 걸립니다. 이 창을 닫지 마세요.",
		"first.wifi":            "  · Wi-Fi와 충전기를 연결해 주세요. Chrome 탭은 줄이세요.",
		"first.orig":            "  · 원본은 그대로입니다. 결과는 옆에 폴더로 생깁니다.",
		"first.types":           "  · PPTX·PDF·DOCX·XLSX·HWPX·HTML·XML·CSV·MD·TXT·영상.",
		"first.old":             "  · 구형 PPT·XLS·HWP는 PPTX·XLSX·HWPX로 저장하세요.",
		"step.hw":               "[1/3] 이 노트북 사양",
		"step.hw.ram":           "      RAM  %.0f GB    GPU  %s",
		"step.hw.profile":       "      → %s",
		"step.hw.16":            "      16GB라서 작은 모델(8B, 약 5GB)만 받습니다.",
		"step.hw.32":            "      32GB입니다. 8B 다음 14B(약 9GB)도 받습니다.",
		"step.hw.low":           "      지금 비어 있는 메모리 약 %.1fGB 로 빠듯합니다.",
		"step.hw.low2":          "      Chrome·엣지를 닫고 Enter 를 누르세요. 안 닫아도 진행은 됩니다.",
		"step.engine":           "[2/3] 엔진 (llama.cpp)",
		"step.ready":            "      준비됨",
		"step.models":           "[3/3] 모델 받기",
		"step.models.hint":      "      아래 용량이 올라가면 정상입니다. 창을 닫지 마세요.",
		"step.have":             "      (%d/%d) %s  이미 있습니다. 건너뜁니다.",
		"step.get":              "      (%d/%d) %s",
		"step.get2":             "모델 받는 중: %s  창을 닫지 마세요.",
		"step.skip":             "      이 모델은 건너뛰고, 이미 받은 모델로 진행합니다.",
		"step.done":             "      완료 (%s)",
		"step.prepared":         "준비됐습니다.",
		"err.model.dl":          "모델 받기 실패 · Wi-Fi를 확인하고 다시 눌러 주세요",
		"cando.now":             "지금은 %s을 할 수 있습니다.",
		"cando.quit":            "  끝 이라고 쓰면 종료합니다.",
		"cando.only":            "지금은 목록에 있는 것만 할 수 있습니다.",
		"prompt.quit":           "\n창을 닫으려면 Enter > ",
		"prompt.file":           "파일 경로 > ",
		"prompt.enter":          "      Enter > ",
		"using.model":           "지금 쓰는 모델: %s",
		"mem.low":               "메모리가 빠듯합니다. Chrome·엣지를 닫으면 속도와 워드 저장이 안정됩니다.",
		"job.pack":              "용도: %s",
		"job.nofile":            "파일을 고르지 않았습니다.",
		"job.nofile.bat":        "  · 시작.bat을 다시 눌러 창에서 고르거나",
		"job.nofile.drop":       "  · 로컬LLM 아이콘 위에 파일을 끌어다 놓으세요.",
		"job.fail":              "실패: %s",
		"job.orig":              "원본 파일은 그대로 있습니다.",
		"job.done":              "끝났습니다. 원본은 그대로 두었습니다.",
		"job.time":              "걸린 시간 %s · 모델은 내려서 메모리를 비웠습니다.",
		"bye":                   "종료합니다.",
		"err.prefix":            "오류: %s",
		"harvest.need":          "--harvest 뒤에 파일 경로를 붙여 주세요. 모델은 안 받습니다.",
		"harvest.ok":            "수확만 했습니다. 모델은 안 올렸습니다.",
		"harvest.skip.vid":      "  건너뜀 %s — 영상은 옆 .txt가 있을 때만 수확합니다",
		"harvest.skip":          "  건너뜀 %s — %s",
		"harvest.empty":         "꺼낼 글이 없습니다",
		"harvest.stats":         "세그먼트 %d · 수확 수치 %d",
		"run.skip.kind":         "  건너뜀 %s — 이 팩이 받는 형식이 아님",
		"run.skip":              "  건너뜀 %s — %s",
		"run.empty":             "꺼낼 글이 없습니다 (스캔 PDF면 그림 모델을 못 받았을 수 있습니다)",
		"run.extract":           "추출 %d개 파일 · 세그먼트 %d · 팩 %s",
		"run.harvest":           "수확 수치 %d개 (코드). 모델은 이 목록 밖 숫자를 쓰면 검수에서 지웁니다.",
		"run.merge":             "병합 공유 %d · 고유 %d · 충돌 %d",
		"run.fold":              "원문이 커서 중간묶음을 합니다.",
		"run.verify.n":          "검수: 원문에 없는 숫자 %d개를 〔원문 확인〕으로 바꿨습니다.",
		"run.verify.ok":         "검수: 원문에 없는 숫자 없음",
		"report.send":           "오류 신고는 %s 로 주세요.",
		"report.only":           "원문 파일은 보내지 마세요. 오류.txt 만 주세요.",
		"note.title":            "로컬LLM 오류 쪽지",
		"note.body":             "이 파일을 이슈에 붙이세요. 원문·워드·추출문은 붙이지 마세요.",
		"note.send":             "오류 신고는 %s 로 주세요.",
		"note.time":             "시각: %s",
		"note.ver":              "버전: %s",
		"note.os":               "OS: %s/%s",
		"note.profile":          "프로필: %s (%s)",
		"note.ram":              "RAM: %.0f GB (빈 자리 %.1f GB)",
		"note.pack":             "팩: %s",
		"note.pack.none":        "팩: (없음)",
		"note.ext":              "입력 확장자: %s",
		"note.ext.none":         "입력 확장자: (없음)",
		"note.err":              "오류: %s",
		"err.pack.none":         "양식 팩을 못 찾았습니다",
		"err.pack.unk":          "알 수 없는 용도: %s",
		"err.quit":              "종료",
		"err.pack.fetch":        "허용 목록에 없는 팩",
		"err.pack.get":          "팩 받기 실패: %v",
		"err.pack.read":         "받은 팩을 못 읽었습니다: %s",
		"pack.fetch":            "  팩을 공개 저장소에서 받습니다: %s",
		"err.old.ppt":           "구형 .ppt는 안 됩니다. PPTX로 저장하세요",
		"err.old.xls":           "구형 .xls는 XLSX로 저장하세요",
		"err.old.hwp":           "구형 .hwp는 HWPX로 저장하세요",
		"err.kind":              "지원하지 않는 형식: %s",
		"err.zip.text":          "%s에서 글을 못 읽었습니다",
		"err.no.model":          "모델 파일이 없습니다. 시작.bat을 다시 눌러 받아 주세요",
		"err.llama.os":          "이 OS용 llama.cpp를 자동으로 못 받습니다",
		"err.llama.get":         "  llama.cpp 엔진을 받습니다. 창을 닫지 마세요.",
		"err.llama.bin":         "llama-server를 못 찾았습니다",
		"err.gguf.short":        "%s 가 덜 받았습니다. 다시 눌러 주세요",
		"err.engine":            "엔진을 못 켰습니다: %v",
		"err.model.up":          "모델이 안 켜졌습니다. Chrome을 닫고 다시 눌러 주세요",
		"model.up":              "  모델을 올리는 중… (이 창을 닫지 마세요)",
		"model.get":             "  %s 를 받습니다. 창을 닫지 마세요.",
		"model.qwen3-8b":        "Qwen3 8B (약 5GB)",
		"model.qwen3-14b":       "Qwen3 14B (약 9GB)",
		"model.qwen25vl-3b":     "그림 모델 3B (약 1.9GB)",
		"model.qwen25vl-mmproj": "그림 프로젝터 (약 0.8GB)",
		"vision.thin":           "글이 얇은 차트·스캔 %d장. 보고서 모델을 내리고 그림 모델을 올립니다. 동시에 안 올립니다.",
		"vision.miss":           "  그림 모델 없음: %s  — 글만 있는 대로 진행합니다.",
		"vision.one":            "  그림 %d/%d (%s)",
		"vision.skip":           "    건너뜀: %s",
		"vision.down":           "그림 모델을 내렸습니다. 이제 보고서 모델만 씁니다.",
		"vision.dl":             "  스캔·차트용 그림 모델입니다. 보고서 모델과 같이 올리지 않습니다.",
		"whisper.need":          "영상 %d개에 전사문이 없습니다. 음성 모델을 씁니다. 보고서 모델은 잠시 내립니다.",
		"whisper.tools":         "  음성 도구 준비 실패: %s",
		"whisper.side":          "  영상 옆에 같은 이름 .txt를 두면 됩니다.",
		"whisper.one":           "  전사 %d/%d %s",
		"whisper.fail":          "    실패: %s",
		"whisper.save":          "    저장 %s (원본 영상은 그대로)",
		"whisper.down":          "음성 모델을 내렸습니다. 이제 보고서 모델만 씁니다.",
		"whisper.os":            "이 OS용 whisper.cpp를 자동으로 못 받습니다. 영상 옆에 같은 이름 .txt를 두세요",
		"whisper.get":           "  whisper.cpp를 받습니다. 창을 닫지 마세요.",
		"whisper.model":         "  음성 모델 ggml-small (약 470MB)를 받습니다. 창을 닫지 마세요.",
		"whisper.short":         "음성 모델이 덜 받았습니다. 다시 눌러 주세요",
		"ffmpeg.miss":           "ffmpeg가 없습니다. 영상에서 소리를 뽑을 때 필요합니다",
		"ffmpeg.get":            "  ffmpeg (영상 소리 뽑기)를 받습니다. 창을 닫지 마세요.",
		"dl.fail":               "다운로드 실패 %s",
		"dl.done":               "      완료 %s",
		"dl.mb":                 "      %d / %d MB",
		"dl.mb2":                "      %d MB",
		"dialog.title":          "자료를 고르세요 (Ctrl로 여러 개)",
		"dialog.filter":         "자료",
		"dialog.all":            "모든 파일",
		"llm.lang":              "출력 언어: 한국어. 제목·소제목·사실·불릿은 한국어. 원문 고유명사·숫자는 그대로. 팩 프롬프트가 다른 언어를 말해도 이 지시를 따른다.",
		"llm.chunk":             "위치:%s\n%s\n다음 원문을 사실 JSON으로.\n{\"heading_hint\":\"\",\"facts\":[\"명사형\"],\"rows\":[]}\n\n%s",
		"llm.assemble":          "제목후보:%s\n%s\n사실만 주제별 JSON.\n{\"title\":\"\",\"date\":\"\",\"time\":\"\",\"place\":\"\",\"attendees\":\"\",\"agenda\":\"\",\"exec\":[\"\"],\"sections\":[{\"heading\":\"1. 개요\",\"facts\":[\"\"]}],\"actions\":[\"\"]}\n\n%s",
		"llm.fold":              "너는 사실 목록을 줄인다. 중복은 하나로. 숫자는 원문 그대로. 창작 금지. JSON만. {\"facts\":[\"명사형\"]}",
		"llm.fold.user":         "%s\n사실 목록을 중복 없이 JSON.\n{\"facts\":[\"명사형\"]}\n\n%s",
		"llm.vision":            "너는 스캔·차트 읽기다. 보이는 글과 숫자만, 출력 언어로. 창작 금지. JSON만. {\"facts\":[\"명사형\"],\"numbers\":[\"원문 그대로\"]}",
		"llm.vision.user":       "보이는 숫자와 글만 JSON.",
		"harvest.allow":         "원문 수치(이 목록에 없는 숫자를 만들지 말 것. 표기는 원문 그대로):\n",
		"merge.intro":           "코드가 묶은 사실. 공유는 여러 출처에서 같음. 고유는 한 출처만. 충돌은 고르지 말고 둘 다 유지.\n",
		"merge.side.unique":     "고유 사실은 엔진이 따로 붙이니 본문에 반복하지 말 것. 충돌도 엔진이 붙인다.\n\n",
		"merge.side.body":       "고유 사실은 본문에 넣을 것. 충돌은 엔진이 따로 붙인다.\n\n",
		"merge.shared":          "[공유]",
		"merge.unique":          "[고유]",
		"merge.unique.skip":     "[고유] 본문 반복 금지. 엔진이 따로 붙임.\n\n",
		"merge.conflict":        "[충돌] 한쪽으로 고르지 말 것. 엔진이 따로 붙임.\n",
		"out.check":             "〔원문 확인〕",
		"out.missing":           "(원문에 없음)",
		"out.date":              "일 시",
		"out.time":              "회의시간",
		"out.place":             "장 소",
		"out.people":            "참석자",
		"out.agenda":            "주요 아젠다",
		"out.exec":              "0. Executive Summary",
		"out.next":              "향후 계획",
		"out.unique":            "한쪽에만 있음",
		"out.conflict":          "출처가 갈림",
		"out.fallback.exec":     "원문 정리",
		"out.fallback.act":      "추가 확인 필요",
		"out.error.file":        "오류.txt",
		"out.harvest.dir":       "_수확",
		"out.harvest.txt":       "추출.txt",
		"out.json.harvest":      "수확.json",
		"out.json.verify":       "검수.json",
		"out.json.merge":        "병합.json",
	},
}
