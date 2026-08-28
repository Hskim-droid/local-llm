# 로컬 보고서 (LG 그램 / 윈도우)

파워셸에 **한 번** 실행하면, 이 노트북 RAM을 보고 **받을 모델을 정해 순서대로 다운**한 뒤, 영상·PPT·PDF를 한글 워드로 뽑습니다. 문서는 노트북 밖으로 나가지 않습니다.

```
영상 mp4   →  음성 전사 (같은 이름 .txt 가 있으면 생략)
PPTX       →  슬라이드 글
PDF        →  페이지 글
           →  로컬 Ollama가 회의록 JSON
           →  report.docx  (워드가 열림)
```

상세 스펙·모델표: [docs/GRAM.ko.md](docs/GRAM.ko.md)

## 그램에서 쓰는 법 (이대로만)

### 1. 준비물 두 개 (최초 1회)

1. **Python 3.11+**  
   https://www.python.org/downloads/windows/  
   설치 화면 **맨 아래 `Add python.exe to PATH` 체크**.
2. **Ollama**  
   https://ollama.com/download  
   설치 후 트레이에 라마 아이콘이 보이면 됩니다.  
   `ollama serve` 창을 띄워 둘 필요는 없습니다.

저장소를 받습니다.

```powershell
git clone https://github.com/Hskim-droid/local-report.git
cd local-report
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
.\report.ps1
```

### 2. 첫 실행이 하는 일 (창을 닫지 마세요)

스크립트가 한국어로 단계를 말해 줍니다.

1. RAM·GPU 조사 → `gram16`(16GB) 또는 `gram32`(32GB)
2. pip 패키지 설치
3. Ollama가 없으면 설치 주소를 알려 주고 Enter를 기다림
4. 모델 순차 다운로드 (진행률이 아래에 그대로 보임)

| 이 그램 | 순서대로 받는 것 | 크기 |
|---|---|---|
| **16GB** (가장 흔함) | `qwen3:8b` 만 | 약 5GB |
| **32GB** | `qwen3:8b` → `qwen3:14b` | 약 5GB + 9GB |
| 14B / 32B / 업스테이지 대형은 **16GB에 안 심음** | | |

Wi-Fi, **충전기 연결**을 권합니다. 끝나면 `준비됐습니다` 가 나옵니다.

### 3. 그다음부터 (보고서 뽑기)

같은 창 또는 새 파워셸:

```text
.\report.ps1
```

`영상 · PPT · PDF 를 이 창에 끌어다 놓고 Enter` 가 뜨면,

1. 탐색기에서 파일 세 개를 고른다 (Ctrl 클릭)
2. 파워셸 창으로 끌어다 놓는다
3. Enter

저장 위치는 **맨 먼저 놓은 파일 옆**:

```text
바탕화면\회의.mp4
바탕화면\회의_report\report.docx
```

원본은 건드리지 않습니다. 워드가 자동으로 열립니다.

PATH에 넣으려면 `.\install-windows.ps1` 후 **새** 파워셸에서 `report` 만 치면 됩니다.

## 자주 쓰는 옵션

```powershell
.\report.ps1 --status              # 사양·받을 모델만 보기 (다운로드 없음)
.\report.ps1 --out $HOME\Desktop\out
.\report.ps1 --model qwen3:8b      # 32GB여도 8B만
.\report.ps1 --force               # 설치를 처음부터 다시 (bootstrap)
```

`--force` 는 `python bootstrap.py --force` 로도 됩니다.

## 모델 왜 이 조합인가

- 그램은 대개 **엔비디아 VRAM이 없고 RAM으로** 모델을 올립니다.
- **16GB**에 14B를 올리면 윈도우·브라우저와 싸워 스왑이 납니다. 그래서 8B를 심습니다.
- **Qwen** 은 일본 장표 → 한국어에 유리합니다. LG **EXAONE 7.8B** 는 한글 문장용이며 기본 자동 다운로드에는 넣지 않았습니다 (RAM을 두 모델이 나눠 쓰면 그램이 힘듭니다).
- 업스테이지 Solar Open 2 는 250B급이라 그램에서 못 돌립니다.

## 맥

```bash
./install-macos.sh
python3 bootstrap.py
./report
```

24GB 맥은 `qwen3:14b` 를 받습니다.
