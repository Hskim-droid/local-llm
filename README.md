# local-llm

**엔진**입니다. Go CLI + 이 노트북의 로컬 모델(llama.cpp GGUF). 파이썬·Ollama·클라우드 API 없음. 파일은 나가지 않습니다.

용도는 앱이 아니라 **`packs/`** 입니다. 프롬프트 세 장만 바꾸면 됩니다. 새 저장소·새 exe를 만들지 마세요. 규약: [packs/README.md](packs/README.md)

```
추출·수확·검수(코드) → 팩이 JSON 채움 → 파일로 출력
채팅·그림·음성 모델은 한 번에 하나만
```

팩 목록은 공개 저장소 `gramapp/packs/index.json` 입니다. 없는 팩은 그때 받습니다. 모델(GGUF)은 Hugging Face에서 받습니다. GitHub에 모델을 올리지 않습니다.

실행하면 한 줄로만 갑니다. 끝나면 다시 “지금은 ~ 할 수 있습니다.” 끝 이라고 쓰면 종료합니다.

지금 들어 있는 팩은 시작용입니다.

| 번호 | 팩 | 결과 |
|---|---|---|
| 1 | 보고서 | `이름_보고서\보고서.docx` |
| 2 | 회의록 | `이름_회의록\회의록.docx` |
| 3 | 일한 번역 | `이름_번역\한국어번역.docx` |

윈도우 실행 파일은 `로컬LLM.exe`. 16GB는 8B만, 32GB는 8B 다음 14B. 숫자는 코드가 원문에서 걷고, 목록 밖은 〔원문 확인〕.

파일은 회사 밖·인터넷으로 올라가지 않습니다. [PRIVACY.md](PRIVACY.md)

스캔 PDF·차트만 있는 슬라이드는 글을 못 뽑을 때만 Qwen2.5-VL 3B GGUF를 **잠깐** 올립니다. 채팅 모델은 그 전에 내립니다.

영상은 같은 이름 `.txt`가 있으면 그걸 씁니다. 없으면 `whisper.cpp` + `ggml-small`(약 470MB)로 전사한 뒤 음성 모델을 내리고 8B를 올립니다. 윈도우·맥 모두 필요할 때 도구를 받습니다. 원본 영상은 그대로, 전사문만 옆에 `.txt`로 저장합니다.

## 그램 — 받는 법 · 더블클릭

받는 법 자세히: **[docs/다운로드.md](docs/다운로드.md)**

1. https://github.com/Hskim-droid/local-llm/releases/latest
2. **Assets** 에서 `local-llm-windows.zip` 받기 (`Source code` 아님)
3. **바탕화면에 압축을 풉니다.** 압축 파일 안에서 바로 실행하지 마세요.
4. 푼 폴더에서 **`시작.bat` 더블클릭**
5. Windows가 막으면 **추가 정보 → 실행**
6. 검은 창의 한글 안내대로 모델을 받습니다. 창을 닫지 마세요.
7. **무엇을 만들까요?** 에서 1 보고서 / 2 회의록 / 3 번역
8. 파일 창에서 영상·PPT·PDF를 고릅니다.

아이콘 위에 파일을 **끌어다 놓아도** 됩니다. 용도는 그다음에 고릅니다.

프로그램은 약 7MB. AI 모델은 첫 실행에 받습니다 (16GB 그램 약 5GB).

모델 없이 원문 숫자만 보려면:

```bash
cd gramapp
go run . --harvest ../examples/slides_jp.txt
```

파이썬 스크립트(`report.ps1`)는 아래 **개발용**이며, 아직 Ollama를 씁니다. 그램 사용자 경로가 아닙니다.

> 예전에 쓰던 문서번역기 앱은 이 저장소의 **번역** 팩으로 이어집니다.

## 그램 (윈도우, 파이썬)

1. [Python](https://www.python.org/downloads/windows/) — 설치 화면에서 **Add python.exe to PATH**
2. [Ollama](https://ollama.com/download) — 트레이에 라마 아이콘
3. 이 폴더에서:

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
.\report.ps1
```

첫 실행 안내 (한국어):

| 이 노트북 | 순서대로 받음 | 용량 |
|---|---|---|
| 그램 **16GB** | `qwen3:8b` | 약 5GB |
| 그램 **32GB** | `qwen3:8b` → `qwen3:14b` | 약 5GB + 9GB |

창을 닫지 마세요. 끝나면 파일을 기다리므로 **같은 창에** 세 파일을 끌어다 놓고 Enter.

산출:

```text
바탕화면\회의.mp4
바탕화면\회의_보고서\보고서.docx
```

끝나면 **폴더가 열립니다.** 워드를 바로 열려면 `--open`. 작업이 끝나면 모델을 내려 RAM을 비웁니다.

Chrome·Safari를 많이 켠 채 돌리면 맥/그램 모두 스왑이 나고, 워드 저장이 실패할 수 있습니다. 실행 전에 브라우저 탭을 줄이세요.

자세한 스펙: [docs/GRAM.ko.md](docs/GRAM.ko.md)

```powershell
.\report.ps1 --status
.\install-windows.ps1    # 이후 새 창에서 report 만
```

## 맥

```bash
./report
```

24GB면 `qwen3:14b`를 받습니다. 산출 폴더 이름은 동일합니다 (`*_보고서/보고서.docx`).

Go 앱(용도 팩 포함):

```bash
cd gramapp
go run . --pack 보고서 자료.pptx
```

## 양식 팩

정본: `gramapp/packs/`. exe 안에 들어가고, **푼 ZIP 옆 `packs/`가 있으면 그게 우선**입니다. 고객·사내용 톤은 이 저장소에 올리지 말고 옆 폴더로만 씁니다.

안내: [packs/README.md](packs/README.md)
