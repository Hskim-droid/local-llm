# local-report

Drop **a video, a PowerPoint, and a PDF** on a terminal. A **local** Ollama model writes a Korean Word report. Nothing is uploaded.

**Primary target: LG gram on Windows.** 16 GB machines stay on an 8B model. 32 GB may use 14B. A 24 GB Mac still works.

```
mp4 / m4a  →  Whisper (or a sibling .txt)
pptx       →  slide text
pdf        →  page text
           →  Ollama fills a minutes JSON (topics grouped, not one section per slide)
           →  report.docx
```

Full gram spec, model table, and click-by-click Windows steps:

- English: [docs/GRAM.md](docs/GRAM.md)
- Korean: [docs/GRAM.ko.md](docs/GRAM.ko.md)

```powershell
report --status          # which profile this PC is
```

## Install (Windows gram)

```powershell
py -3 -m pip install -r requirements.txt
ollama pull qwen3:8b          # ~5 GB, correct default for 16 GB
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
.\install-windows.ps1
```

New PowerShell window → type `report` → space → drop three files → Enter.

Output: `{first-file-name}_report\report.docx` next to the first dropped file.

## Install (macOS)

```bash
pip3 install -r requirements.txt
pip3 install mlx-whisper av
ollama pull qwen3:14b         # 24 GB unified
./install-macos.sh
```

## Models (short)

| RAM | Default pull | Optional | Do not pull |
|---|---|---|---|
| gram **16 GB** | `qwen3:8b` | `exaone3.5:7.8b` (Korean tone) | 14B / 32B |
| gram **32 GB** | `qwen3:8b` first, then `qwen3:14b` if you want | EXAONE 7.8B | EXAONE 32B, Solar Open 2 |
| Mac **24 GB** | `qwen3:14b` | 8B for speed | 32B Q4 |

Japanese slides → Korean report: **Qwen**. Korean-only polish: **EXAONE 7.8B**. Upstage Solar Open 2 does not fit a gram.

## Tests

```bash
python3 test_reportctl.py
python3 -c "from hardware import inspect; from ollama_client import load_config; print(inspect(load_config()).summary())"
```

## Privacy

Disk only, plus `http://127.0.0.1:11434`. Optional `ollama pull`.
