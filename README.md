# local-llm

**Engine:** Go CLI + llama.cpp GGUF on this machine. No Python, no Ollama, no cloud. Files stay here.

Jobs are **`packs/`**, not new apps. Contract: [packs/README.md](packs/README.md)

```
extract → harvest → merge → verify (engine)
pack fills JSON → Word
one model at a time: chat / vision / whisper
```

## Windows (LG gram)

Send this link: **https://hskim-droid.github.io/local-llm/**

The page shows the folder, the black window, a finished run, and the Word file, then a PowerShell copy box.

The page is one screen. Copy the PowerShell, paste, then double-click `시작.bat` in the folder that opens. It does not run the program for you.

If GitHub Pages is not on yet, zip: https://github.com/Hskim-droid/local-llm/releases/latest → Assets → **`local-llm-windows.zip`** (not Source code). Extract to the desktop. Do not run from inside the zip. Double-click **`시작.bat`**. If Windows blocks it: **More info → Run anyway**.

First run downloads the model. Leave the window open. Then pick Report / Minutes / Translation (1 / 2 / 3).

You can also drop files onto the exe, then pick the job. Several files are always run **one at a time** — each file gets its own output folder. Nothing is merged across files.

16 GB RAM: 8B only (~5 GB download). 32 GB: 8B then 14B. Numbers come from source text; anything else becomes 〔원문 확인〕.

UI **and Word output** are English by default. Korean Windows UI → Korean UI and Korean Word. Pin both with `--lang ko` or `--lang en` (saved). The translation pack still follows an explicit target language if you state one.

Files never leave the PC. [PRIVACY.md](PRIVACY.md)

Send errors to [GitHub Issues](https://github.com/Hskim-droid/local-llm/issues). Attach only `오류.txt`, never the source documents.

Full download notes (Korean): [docs/다운로드.md](docs/다운로드.md)

## Mac / from source

```bash
cd gramapp
go run . --pack 보고서 deck.pptx
```

24 GB Apple Silicon uses 14B. Same output folders (`*_보고서/보고서.docx`).

Numbers only, no model:

```bash
cd gramapp
go run . --harvest ../examples/slides_jp.txt
```

## Packs

Canonical source: `gramapp/packs/`. Embedded in the exe. A `packs/` folder next to the unzipped exe wins.

Do not put customer tone in this repo. Put it beside the exe.

| Pack | Output (English UI) | Output (Korean UI) |
|---|---|---|
| Report / 보고서 | `name_report\report.docx` | `이름_보고서\보고서.docx` |
| Minutes / 회의록 | `name_minutes\minutes.docx` | `이름_회의록\회의록.docx` |
| Translation / 번역 | `name_translation\translation.docx` | `이름_번역\번역.docx` |

Hardware notes: [docs/GRAM.ko.md](docs/GRAM.ko.md). RAM/OS knobs: [docs/MACHINE.md](docs/MACHINE.md) (`gramapp/machine.json`). First run is a local wizard: it reads RAM, picks a profile, pulls **packs + this JSON from GitHub**, **GGUF from Hugging Face**. Weights are not stored in this repo.

## Legacy

`report.ps1` / `./report` still talk to Ollama. That is not the gram product. Do not send that path to users.
