# 로컬LLM보고서 — LG gram (Windows)

This tool is tuned for **LG gram notebooks on Windows**, not gaming laptops. A gram is thin, cool, and usually has **no NVIDIA VRAM**. Local Ollama therefore runs from **system RAM** (plus a weak iGPU). That is slower than an Apple Silicon 24 GB Mac, and **smaller models win**.

`report --status` prints the profile this PC will use.

## Which gram do you have?

Open **Settings → System → About** (or Task Manager → Performance → Memory).

| Typical SKU | RAM | Graphics | This script’s profile | What actually runs well |
|---|---|---|---|---|
| gram 14/16, most 2024–2026 units | **16 GB** soldered | Intel Graphics / Arc / Radeon 840M | `gram16` | **8B class only** |
| gram / gram Pro with 32 GB | **32 GB** soldered | Intel Arc (no RTX) | `gram32` | 8B comfortable, **14B possible but slow** |
| gram Pro 16/17 **with RTX 5050 8 GB** | 32 GB + 8 GB GDDR | NVIDIA | `gram32` | 8B on GPU, 14B spills into RAM |
| This author’s MacBook Pro M4 Pro | 24 GB unified | Metal 16-core | `mac24` | 14B Q4 is the smooth ceiling |

RAM on gram is **on-board**. You cannot add a stick later. If you buy for this workflow, take **32 GB**.

Windows itself is fine. The limit is RAM + thermals, not “Windows vs Mac”.

## Models (Ollama tags)

Do **not** put two chat models in RAM at once. `OLLAMA_MAX_LOADED_MODELS=1`.

| Tag | On-disk (Q4) | Korean | Japanese slides | gram 16 GB | gram 32 GB |
|---|---|---|---|---|---|
| **`qwen3:8b`** (default first download) | ~5 GB | Good | **Best of this size** | **Use this** | Fast daily driver |
| **`exaone3.5:7.8b`** (LG) | ~4.8 GB | **Best Korean tone** | Weaker than Qwen | Optional polish | Optional |
| `qwen3:14b` | ~9.3 GB | Strong | Strong | **No** (swap) | OK if you already pulled it |
| `exaone3.5:32b` | ~19 GB | Strong Korean | Heavy | No | No |
| Upstage **Solar Open 2** (250B MoE) | tens of GB | Excellent | n/a | **Cannot run** | Cannot run |
| Ollama `solar` 10.7B | ~6 GB | Old, English-leaning | Weak | Skip | Skip |

**First run on a gram always downloads `qwen3:8b`**, even on 32 GB, so the machine stays usable. To use 14B later:

```powershell
ollama pull qwen3:14b
report --model qwen3:14b
```

Korean-only minutes (no Japanese source):

```powershell
ollama pull exaone3.5:7.8b
report --model exaone3.5:7.8b
```

Japanese deck + Korean report: keep **Qwen**. EXAONE is the Korean rewrite, not the JP translator.

## Context and speed (what we change for gram)

| Profile | `num_ctx` | chunk size | `num_predict` | Why |
|---|---|---|---|---|
| gram16 | 4096 | 1000 chars | 800 | 8B + Windows + Chrome must still fit |
| gram32 | 6144 | 1400 | 1100 | 14B if loaded, otherwise 8B with more room |
| mac24 | 8192 | 1600 | 1400 | Unified memory |

Expect roughly:

- **gram 16 GB, Qwen 8B:** several seconds per slide, a 20-slide deck in minutes, not seconds.
- **gram 32 GB, Qwen 14B:** similar to a 24 GB Mac but **hotter and slower**; fan-up is normal.
- **Whisper on video:** `faster-whisper` `small` on CPU. A 30-minute meeting can take longer than the LLM pass. If you already have `meeting.txt` next to `meeting.mp4`, transcription is skipped.

Close Chrome / Edge / Teams before a long job. One 14B weights file plus a browser will swap on 16 GB.

## One-time install (Windows)

### 1. Python 3.11 or 3.12

https://www.python.org/downloads/windows/

Tick **“Add python.exe to PATH”**. In a new PowerShell:

```powershell
py -3 --version
```

### 2. Ollama for Windows

https://ollama.com/download

Installer adds `ollama` to PATH and starts the app in the tray. You do **not** need to leave `ollama serve` in a window. This script starts the server if it is down.

Optional, once, in PowerShell (User env):

```powershell
[Environment]::SetEnvironmentVariable("OLLAMA_MAX_LOADED_MODELS", "1", "User")
[Environment]::SetEnvironmentVariable("OLLAMA_NUM_PARALLEL", "1", "User")
[Environment]::SetEnvironmentVariable("OLLAMA_KEEP_ALIVE", "5m", "User")
```

Log out or reboot so new processes see them.

### 3. FFmpeg (only if you drop videos)

https://www.gyan.dev/ffmpeg/builds/ — or `winget install Gyan.FFmpeg`

### 4. This repo

```powershell
cd $HOME\Downloads
git clone https://github.com/Hskim-droid/local-llm-report.git
cd local-llm-report

py -3 -m pip install -r requirements.txt
# first model (~5 GB). Stay on Wi-Fi.
ollama pull qwen3:8b

Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
.\install-windows.ps1
```

`install-windows.ps1` copies the scripts to `%USERPROFILE%\bin` and prepends that folder to your **user** PATH.

Open a **new** PowerShell window (PATH only refreshes for new sessions).

```powershell
report --status
```

You want to see `gram16` or `gram32` and `default qwen3:8b` (or 14B listed under prefer if it is already installed).

## Every time (drop three files)

1. Save the three sources on disk (Desktop is fine).
2. Open PowerShell.
3. Type `report` and a **space**.
4. In Explorer, select the **video**, **PPTX**, and **PDF** (Ctrl-click). Drag them onto the PowerShell window.
5. Press **Enter**.

Same thing from the repo folder, no PATH:

```powershell
cd path\to\local-llm-report
.\report.ps1
```

Then drop files, Enter.

If you press Enter with no files, the script waits: drop, then Enter again.

### What happens

```text
extracting 3 file(s)…
  N segment(s)
LG gram 16 GB (typical Windows)  RAM 16 GB  GPU Intel Graphics  default qwen3:8b
model qwen3:8b  ctx=4096
  chunk 1/N
  ...
writing Word…
  C:\Users\you\Desktop\meeting_보고서\보고서.docx
```

Word opens. Original files are never overwritten.

### Output location

Next to the **first** dropped file:

```text
C:\Users\you\Desktop\kickoff.mp4
C:\Users\you\Desktop\kickoff_보고서\보고서.docx   ← deliverable
C:\Users\you\Desktop\kickoff_보고서\보고서.json
```

Put the output somewhere else:

```powershell
report --out $HOME\Desktop\out
```

(then drop the three files)

## Flags you will actually use

```powershell
report --status
report --profile gram16          # force the 16 GB limits
report --profile gram32
report --model qwen3:8b
report --model exaone3.5:7.8b
report --model qwen3:14b         # 32 GB only
report --no-llm                  # extract + empty skeleton, no GPU/RAM hit
report --open                 # also open Word (default: reveal folder only)
report --no-pull                 # fail if the model is missing, do not download
```

## Video notes on gram

- First time: `faster-whisper` `small` downloads a small CPU model.
- Prefer a **sidecar** `name.txt` if you already transcribed (Clova, Whisper, human notes). Same stem as the video.
- Skip 1-hour recordings on 16 GB until you have a `.txt`. Use the PPTX + PDF only.

## If it is slow or swapping

1. `report --status` — if it picked `gram32` on a 16 GB machine, something is wrong with RAM read; force `--profile gram16`.
2. `ollama ps` — only **one** model listed.
3. Quit Edge/Chrome.
4. Do not `ollama pull` 14B or 32B on 16 GB.
5. Plug in AC power. gram will throttle on battery.

## Privacy

Files never leave the PC. The only HTTP calls are `http://127.0.0.1:11434` (Ollama). The only optional download is `ollama pull`.
