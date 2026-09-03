# Machine profiles (install wizard)

The exe measures **total RAM, free RAM (load), and OS on the PC**. It does not send that to us. It then downloads only what that snapshot needs.

Each **job** re-reads free RAM. A 32 GB machine with Chrome eating memory is treated like 16 GB (8B, smaller ctx) until memory is free again.

| What | From where | Why |
|---|---|---|
| Engine zip, packs, this JSON | **GitHub** (`Hskim-droid/local-llm`) | Small. Can retune without a new exe. |
| llama.cpp binary | GitHub **ggml-org** releases (OS zip) | CPU Windows / Metal macOS. Field `llama_rel` (e.g. `b10670`). |
| Chat / vision GGUF | **Hugging Face** | 5–9 GB. GitHub cannot host these. |
| Whisper `ggml-small` | Hugging Face | Loaded only for video, then unloaded. |

Never load chat + vision + whisper at once.

## Profiles (`gramapp/machine.json`)

First matching row wins. Numbers are **as shipped** (Gram 16 GB / Mac 24 GB measured 2026-08).

| id | When | Weights | ctx | chunk | predict | threads |
|---|---|---|---|---|---|---|
| mac24 | macOS, 20–28 GB **and** ≥ 8 GB free | 14B Q4 | 8192 | 1600 | 1400 | ≤8 |
| gram32 | ≥ 20 GB **and** ≥ 8 GB free | 8B then 14B | 6144 | 1400 | 1100 | ≤8 |
| gram16 | default | 8B only | 4096 | 1000 | 800 | ≤4 |

**Load (`derate`)** — first rule whose `avail_lt` is above current free RAM wins.

| free RAM | effect |
|---|---|
| &lt; 4 GB | 8B, ctx 2048, predict 512, 2 threads |
| &lt; 6 GB | 8B, ctx 4096, predict 800, 4 threads |
| &lt; 10 GB | 8B, ctx 4096, predict 900, 4 threads |

- **ctx** — llama.cpp `-c`. Prompt + output cap. Raising this on 16 GB swaps.
- **chunk** — characters of source per LLM call before assemble.
- **predict** — `max_tokens`. 800 on 16 GB can truncate a long report JSON; the Word then looks empty. Do not “fix” that by raising ctx first.
- **ngl** — 99 on Darwin (Metal), 0 on Windows CPU zip. Speed, not wording quality.

Temperature stays **0.1**. Quantization stays **Q4_K_M**.

## Tune without a new binary

Edit `gramapp/machine.json` on `main`. First run (and later runs with network) pull it from:

`https://raw.githubusercontent.com/Hskim-droid/local-llm/main/gramapp/machine.json`

If GitHub is down, the copy **embedded in the exe** is used. A `machine.json` next to the exe wins over the network.

Do not put GGUF files in Releases. Point at Hugging Face URLs in code (`llama.go`).

## What the wizard prints

RAM, GPU name, profile id, then ctx/chunk/predict, then which GGUF will be fetched. Files never leave the PC.
