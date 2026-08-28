# 로컬 보고서

파워셸(또는 맥 터미널)에 영상·PPT·PDF를 끌어다 놓으면 **이 노트북에서만** 한글 워드 보고서를 만듭니다.

LG 그램 16/32GB 윈도우를 기준으로, 첫 실행이 RAM을 보고 **받을 모델을 정해 순서대로 다운**합니다.

## 그램 (윈도우)

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
