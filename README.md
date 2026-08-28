# local-report

Drop **a video, a PowerPoint, and a PDF** onto your terminal. A local Ollama model (14B) writes a clean Korean Word report. Nothing is uploaded.

```
mp4 / m4a  →  speech-to-text (Whisper; reuses a sibling .txt if present)
pptx       →  slide text
pdf        →  page text
           →  Ollama fills a minutes JSON
           →  report.docx
```

This is a **dev script**, not an installer app. You need Python and [Ollama](https://ollama.com) on the same machine.

## What you get

Next to the **first** file you dropped:

```text
meeting_report/
  report.docx    ← the deliverable, then it opens
  report.json    ← regenerate later; not for sharing
```

The Word file is plain text: title, metadata, bullets, light tables. Korean body copy uses Malgun Gothic on Windows and Nanum Gothic / Apple SD Gothic on Mac.

## macOS

```bash
# once
brew install python ffmpeg
pip3 install -r requirements.txt
pip3 install mlx-whisper av          # fast transcription on Apple Silicon
# install Ollama from https://ollama.com then:
ollama pull qwen3:14b

chmod +x install-macos.sh report
./install-macos.sh
```

Then in a **new** Terminal window:

1. Type `report` and a space
2. Drag a video, a `.pptx`, and a `.pdf` from Finder onto the window
3. Press Enter

If you only type `report` and press Enter, the script waits for the drop.

## Windows (PowerShell)

```powershell
# once — Python 3.11+ from python.org, tick "Add python.exe to PATH"
py -m pip install -r requirements.txt
# install Ollama from https://ollama.com then:
ollama pull qwen3:14b

Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
.\install-windows.ps1
```

Open a **new** PowerShell window:

1. Type `report` and a space
2. Drag a video, a `.pptx`, and a `.pdf` from Explorer onto the window
3. Press Enter

From the repo folder, without installing: `.\report.ps1` or `.\report.cmd`.

## Hardware

A 24 GB Apple Silicon Mac runs `qwen3:14b` comfortably. On 16 GB RAM use `qwen3:8b`:

```bash
ollama pull qwen3:8b
report --model qwen3:8b
```

The script starts `ollama serve` if it is not running, and downloads the preferred model if none is installed (several GB, once).

## Inputs

| Kind | Extensions | Notes |
|---|---|---|
| Video / audio | `.mp4` `.m4a` `.mov` `.wav` `.mp3` | If `meeting.txt` sits next to `meeting.mp4`, transcription is skipped |
| Slides | `.pptx` | Old `.ppt` is not supported — save as PPTX |
| Document | `.pdf` | Text PDFs work; scanned pages need extra OCR |

About three files at a time is the intended use. Japanese and English are translated **directly into Korean** (no English pivot).

## Useful flags

```bash
report meeting.mp4 deck.pptx brief.pdf
report --out ~/Desktop/out
report --no-llm          # extract + skeleton only, no model
report --no-open         # write the docx but do not open it
report --model qwen3:8b
```

## Tests

```bash
python3 test_reportctl.py
```

No Ollama required. Video tests use a sidecar `.txt`, so Whisper is not downloaded.

## Privacy

Files stay on disk. The only network the script uses is `http://127.0.0.1:11434` (local Ollama). Optional one-time `ollama pull` is the only download.
