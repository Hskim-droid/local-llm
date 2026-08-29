# local-llm

**Engine:** Go CLI + llama.cpp GGUF on this machine. No Python, no Ollama, no cloud. Files stay here.

Jobs are **`packs/`**, not new apps. Contract: [packs/README.md](packs/README.md)

```
extract → harvest → merge → verify (engine)
pack fills JSON → Word
one model at a time: chat / vision / whisper
```

## Windows (LG gram)

1. https://github.com/Hskim-droid/local-llm/releases/latest
2. Assets → **`local-llm-windows.zip`** (not Source code)
3. Extract to the desktop. Do not run from inside the zip.
4. Double-click **`시작.bat`**
5. If Windows blocks it: **More info → Run anyway**
6. First run downloads the model. Leave the window open.
7. The program says **지금은 보고서, 회의록, 번역을 할 수 있습니다.** Pick 1 / 2 / 3.
8. Choose PPTX, PDF, DOCX, XLSX, HWPX, HTML, XML, CSV, MD, TXT, or video.

You can also drop files onto the exe, then pick the job.

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

Hardware notes: [docs/GRAM.ko.md](docs/GRAM.ko.md)

## Legacy

`report.ps1` / `./report` still talk to Ollama. That is not the gram product. Do not send that path to users.
