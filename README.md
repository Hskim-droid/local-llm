# 로컬LLM보고서

그램에서 **더블클릭** 한 번으로, 영상·PPT·PDF를 **이 노트북에서만** 한글 워드로 만듭니다. Go로 만든 작은 실행 파일이라 파이썬을 깔 필요가 없습니다.

실행 파일은 하나입니다. 용도만 고릅니다.

| 번호 | 팩 | 결과 폴더 |
|---|---|---|
| 1 | 보고서 | `이름_보고서\보고서.docx` |
| 2 | 회의록 | `이름_회의록\회의록.docx` |
| 3 | 일한 번역 | `이름_번역\한국어번역.docx` |

첫 실행이 RAM을 보고 모델을 **순서대로 받습니다.** (16GB → 8B만, 32GB → 8B 다음 14B)

파일은 회사 밖·인터넷으로 올라가지 않습니다. [PRIVACY.md](PRIVACY.md)

숫자는 코드가 원문에서 먼저 걷고, 모델이 목록 밖 숫자를 쓰면 **〔원문 확인〕**으로 바꿉니다. 그램 RAM을 더 쓰지 않습니다.

스캔 PDF·차트만 있는 슬라이드는 글을 못 뽑을 때만 `qwen2.5vl:3b`(약 3.2GB)를 **잠깐** 올립니다. 보고서 모델은 그 전에 내립니다.

## 그램 — 받는 법 · 더블클릭

받는 법 자세히: **[docs/다운로드.md](docs/다운로드.md)**

1. https://github.com/Hskim-droid/local-llm-report/releases/latest
2. **Assets** 에서 `local-llm-report-windows.zip` 받기 (`Source code` 아님)
3. **바탕화면에 압축을 풉니다.** 압축 파일 안에서 바로 실행하지 마세요.
4. 푼 폴더에서 **`시작.bat` 더블클릭**
5. Windows가 막으면 **추가 정보 → 실행**
6. 검은 창의 한글 안내대로 모델을 받습니다. 창을 닫지 마세요.
7. **무엇을 만들까요?** 에서 1 보고서 / 2 회의록 / 3 번역
8. 파일 창에서 영상·PPT·PDF를 고릅니다.

아이콘 위에 파일을 **끌어다 놓아도** 됩니다. 용도는 그다음에 고릅니다.

프로그램은 약 7MB. AI 모델은 첫 실행에 받습니다 (16GB 그램 약 5GB).

파이썬 스크립트(`report.ps1`)는 아래 **개발용**입니다.

> 예전에 쓰던 [Munseo Translator](https://github.com/Hskim-droid/munseo-translator) (문서번역기)는 이 저장소의 **번역** 팩으로 이어집니다. 그쪽은 보관 상태입니다.

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

`gramapp/packs/` — `pack.json` + `chunk.txt` + `assemble.txt`. exe 안에 들어가고, exe 옆 `packs/`가 있으면 그게 우선입니다. 안내: [packs/README.md](packs/README.md)
