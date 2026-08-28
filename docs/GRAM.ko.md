# LG 그램 (윈도우) — 스펙, 모델, 사용법

이 스크립트는 **윈도우 LG 그램** 기준으로 맞춰 두었습니다. 그램은 얇고, 대개 **엔비디아 VRAM이 없습니다.** Ollama는 **시스템 RAM**으로 돌아갑니다. 지금 쓰는 맥북 프로 M4 Pro 24GB보다 로컬 LLM은 약하고, **작은 모델이 정답**입니다.

`report --status` 로 이 PC가 어떤 프로필인지 확인하세요.

## 내 그램이 어디인가

설정 → 시스템 → 정보, 또는 작업 관리자 → 성능 → 메모리.

| 흔한 구성 | RAM | 그래픽 | 프로필 | 실제로 되는 것 |
|---|---|---|---|---|
| 그램 14/16, 2024–2026 다수 | **16GB** (온보드) | 인텔/Arc/라데온 iGPU | `gram16` | **8B만** |
| 그램·그램 프로 32GB | **32GB** | 인텔 Arc (RTX 없음) | `gram32` | 8B 편함, **14B는 가능하나 느림** |
| 그램 프로 + **RTX 5050 8GB** | 32GB + 8GB | 엔비디아 | `gram32` | 8B는 GPU, 14B는 RAM으로 넘침 |
| (참고) 맥북 프로 M4 Pro | 24GB 통합 | Metal | `mac24` | 14B Q4가 원활 상한 |

그램 RAM은 **온보드**라 나중에 못 답니다. 이 용도로 사면 **32GB**.

윈도우라서가 아니라 **램·발열**이 한계입니다.

## 모델

채팅 모델은 **한 개만** 올립니다.

| 태그 | 디스크 | 한국어 | 일본 장표 | 그램 16GB | 그램 32GB |
|---|---|---|---|---|---|
| **`qwen3:8b`** (첫 다운로드) | 약 5GB | 무난 | **이 크기에서 제일 나음** | **기본** | 빠른 일상 |
| **`exaone3.5:7.8b`** (LG) | 약 4.8GB | **문장이 가장 자연스러움** | Qwen보다 약함 | 선택 | 선택 |
| `qwen3:14b` | 약 9.3GB | 좋음 | 좋음 | **금지 (스왑)** | 받아 둔 경우만 |
| `exaone3.5:32b` | 약 19GB | 좋음 | 무거움 | 불가 | 불가 |
| 업스테이지 **Solar Open 2** | 초대형 | 좋음 | — | **로컬 불가** | 불가 |

그램은 **처음엔 무조건 `qwen3:8b`** 를 받습니다. 32GB여도 첫 실행을 가볍게 하려는 것입니다. 14B를 쓰려면:

```powershell
ollama pull qwen3:14b
report --model qwen3:14b
```

일본 장표 → 한글 보고서는 **Qwen**. 한글만 다듬을 때 EXAONE.

## 한 번만 설치

1. Python 3.11+ https://www.python.org/downloads/windows/  
   **Add python.exe to PATH** 체크. `py -3 --version`
2. Ollama https://ollama.com/download  
   트레이에 뜨면 됩니다. `ollama serve` 창을 띄워 둘 필요 없음.
3. 영상 넣을 거면 FFmpeg (`winget install Gyan.FFmpeg`)
4. 이 저장소:

```powershell
git clone https://github.com/Hskim-droid/local-report.git
cd local-report
py -3 -m pip install -r requirements.txt
ollama pull qwen3:8b
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
.\install-windows.ps1
```

**새** 파워셸을 연 다음:

```powershell
report --status
```

`gram16` 또는 `gram32`, `default qwen3:8b` 가 보이면 됩니다.

## 매번

1. 영상·PPT·PDF를 디스크에 둔다.
2. 파워셸에 `report` 치고 **스페이스**.
3. 탐색기에서 세 파일을 끌어다 놓는다.
4. Enter.

끝나면 첫 파일 옆에 `이름_report\report.docx` 가 생기고 워드가 열립니다. 원본은 안 건드립니다.

```powershell
report --out $HOME\Desktop\out
```

브라우저·팀즈를 줄이고, 16GB에서는 14B를 받지 마세요. 배터리보다 어댑터.
