# local-llm

A public project by [Hosang Kim](https://github.com/Hskim-droid), built with Codex. I describe the tasks and desired outputs; Codex handles the coding.

[Project context and scope](https://dropkit-contents.pages.dev/work/local-llm/) · [About / dropkit](https://dropkit-contents.pages.dev/about/) · [LinkedIn](https://www.linkedin.com/in/hosang-kim-a0b5a0370/)

The project introduction distinguishes documented features from independently reproduced results. Setup instructions follow below.

> **Positioning:** local-first, draft-first UI-to-document automation framework.
> Inputs stay behind explicit local adapters; outputs are one evidence-linked
> DOCX, XLSX, or PPTX draft.

The complete input/output contract is documented in
[`docs/WORKFLOW_CONTRACT.md`](docs/WORKFLOW_CONTRACT.md).

**Engine:** The Go CLI uses llama.cpp GGUF on this machine. That engine lane uses no Python, no Ollama, and no cloud service; files stay here. The repository also contains a separate experimental Python UI-to-document harness under `agent-harness/`.

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

The Windows ZIP is the engine release lane. The experimental `agent-harness/`
and thin SDK are source-level framework work and are not included in that ZIP
or implied by the latest engine release.

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

## Integrated UI-to-document harness

The same code repository also contains the experimental, draft-first UI-to-document
harness at [`agent-harness/`](agent-harness/README.md). It is the shared control
plane for local browser or native UI observation, queueing, optional loopback
translation, and exactly one DOCX/XLSX/PPTX draft. It does not ship a production
ERP/QMS connector, mail sender, browser profile, credential, or customer data.
The thin SDK in `agent-harness/workflow_api.py` accepts a `WorkflowRequest` and a
local `SourceAdapter`, then returns a verified `DraftArtifact`; adapters keep
source-specific access outside the core.

Run its fixture checks from the repository root:

```bash
python3 -m pip install -r agent-harness/requirements-fixture.txt
python3 -m playwright install chromium
python3 -m unittest discover -s agent-harness/tests -p 'test_*.py'
```

The publication and project notes remain in the separate
[`dropkit.contents`](https://github.com/Hskim-droid/dropkit.contents) repository.
The harness design references are collected in
[`docs/RPA_REFERENCE_LINEAGE.md`](docs/RPA_REFERENCE_LINEAGE.md).

## Legacy

`report.ps1` / `./report` still talk to Ollama. That is not the gram product. Do not send that path to users.
