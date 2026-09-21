# dropkit-agent-harness

> **Status: experimental proof.** This repository is a standalone software
> project, not a production RPA distribution. The
> reproducible public path is synthetic UI → optional loopback translation → one
> DOCX/XLSX/PPTX artifact. Real ERP/QMS, mail, scheduler, unattended writes, and
> host-specific permissions are outside the verified boundary.

This is a small, cross-platform control plane for the proposed “한결” workflow:

```text
scheduled run or request email
  → validated queue item
  → dedicated UI-read session for ERP/QMS
  → evidence and data checks
  → one artifact: xlsx, docx, or pptx
  → draft or approved delivery
  → event, cost, and outcome log
```

The public repository intentionally contains no ERP/QMS URL, browser cookie,
mail credential, screenshot, or sender token. A local adapter is configured on
each MacBook or LG Gram. The same Python queue works on macOS and Windows.

## Start a Codex session

macOS:

```bash
bash scripts/codex-start.sh
```

Windows / LG Gram:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/codex-start.ps1
```

These commands only initialize local state, run a safety check, and record the
session. A repository cannot force Codex to execute arbitrary code merely by
being opened; the launcher or an explicit instruction in the user's working
environment is the attachment point.

## Queue a safe draft job

```bash
python harness.py enqueue \
  --persona han-gyeol \
  --source-system QMS \
  --request "전일 미처리 품질 이슈 일일보고" \
  --output-format docx \
  --recipient quality@example.com

python harness.py run-once --dry-run
python harness.py status
```

`run-once --dry-run` never opens a browser, changes a remote system, creates a
real document, or sends mail. Without a configured local extractor, a live run
is blocked rather than guessed through.

Every enqueue has an idempotency key. Pass a scheduler or mail message key with
`--idempotency-key`; otherwise the harness derives one from the request. A
duplicate delivery returns the original job instead of creating a second one.
Include the business date in a scheduled request or key when the same report
must run again tomorrow.

The public harness does not execute a configured command. A normal live
`run-once` stays blocked until an adapter is explicitly claimed; the local
worker then uses `run-once --claim` to claim the job and receives a handoff.
The adapter must write one local evidence manifest and call:

```bash
python harness.py record-result \
  --job-id JOB_ID \
  --manifest runtime/manifest.json
```

`record-result` verifies the job and source system, requires every declared
check to pass, keeps the artifact below the configured artifact directory,
checks its SHA-256, and only then moves the job to `drafted`. A stale worker can
be recovered with `python harness.py recover --age-seconds 900`.

## Host mapping and local translation

Run the host probe before installing anything:

```bash
python3 bootstrap.py report
python3 bootstrap.py plan --profile all
```

The report contains only operating-system, Python, package-manager, browser,
module, renderer, and capability facts. It does not collect a hostname, username,
credentials, cookies, or a desktop-wide window list. `apply` accepts only the
repository's fixture requirements, Playwright Chromium, or the matching
optional macOS/Windows binding, and requires the explicit
`--allow-network` flag:

```bash
python3 bootstrap.py apply --profile browser --allow-network
```

The plan may include the local absolute path to the fixture requirements; redact
that path before sharing a plan outside the machine.

After a UI adapter returns records, the optional translation stage runs locally
and keeps source and translated values together:

```bash
OLLAMA_NO_CLOUD=1 python3 translation_pipeline.py \
  --input records.json --output translated.json \
  --from ko --to en --fields title,status,owner \
  --backend ollama --model <local-model>
```

The Ollama-compatible adapter accepts only a loopback endpoint, disables proxy
and redirect handling, and requires the Ollama server to be started with
`OLLAMA_NO_CLOUD=1` and restarted before use. The client applies the same
environment guard and rejects cloud-tagged model names, but cannot verify the
configuration of an already-running server. The manifest records loopback
transport separately from the model server's execution location. It fails
closed on a malformed response and has no client-side cloud fallback. The fixture can
exercise the same stage without a model using `--translate-to en
--translation-backend passthrough`. A document renderer consumes the translated
view while the manifest retains the original records and per-field translation
evidence. This keeps UI extraction, translation, and DOCX/XLSX/PPTX rendering as
separate replaceable stages.

## User-selected output format

The queue already carries one explicit `output_format`; the local renderer now
honors that value for all three formats:

```bash
python3 harness.py enqueue \
  --source-system QMS \
  --request "전일 미처리 품질 이슈" \
  --output-format xlsx
```

`document_renderers.py` uses the same normalized columns and records for DOCX,
XLSX, and PPTX. Each output has a format-specific reopen check for headers and
record cells before its manifest can be recorded. XLSX also receives an
`Evidence` sheet; PPTX receives a summary slide, a records table, and evidence
references. A job produces exactly the requested format, never a second hidden
copy. The renderer is a local artifact stage: it does not write back to the UI
or send mail.

The PPTX renderer currently produces a basic summary slide and table slide. It
does not yet paginate large datasets, solve text overflow for long values, or
perform an app-level visual comparison; those are template and host validation
work after the format contract is proven.

The manifest shape is deliberately small and adapter-neutral:

```json
{
  "schema_version": 1,
  "job_id": "job-...",
  "source_system": "QMS",
  "captured_at": "2026-09-21T09:00:00+00:00",
  "records": [{"record_id": "Q-123", "source_ref": "qms://issue/Q-123"}],
  "checks": [
    {"name": "freshness", "passed": true},
    {"name": "duplicate_ids", "passed": true}
  ],
  "artifact": {
    "path": "runtime/artifacts/daily.docx",
    "format": "docx",
    "sha256": "..."
  }
}
```

## Local adapter contract

Set `executor.extract`, `executor.render`, and `executor.send` only in a local
`config.json`, which is ignored by git. The adapters should:

1. use a dedicated, least-privileged account and isolated browser profile;
2. read only the allowed ERP/QMS screens;
3. return structured records plus source timestamps and evidence references;
4. render exactly one requested format;
5. require a human approval gate before any write or external send.

Email is an input channel, not an instruction authority. Validate sender,
subject, attachment type, and recipient allowlist before enqueueing a job.

An empty `recipient_allowlist` means “no recipient restriction at queue time”
for local draft work. It does not enable sending; the public harness has no
sender implementation. A live sender must require an explicit allowlist and a
human approval gate.

## Safety defaults

- persona `han-gyeol` is read/search/draft only;
- default mode is `draft_only`;
- output format is an explicit enum (`xlsx`, `docx`, `pptx`);
- queue order is priority first, then creation time;
- duplicate delivery is suppressed by an idempotency key;
- result manifests and artifact hashes are checked before `drafted`;
- stale `running` jobs can be requeued after a worker crash;
- local state lives under `runtime/` and is not public;
- no live executor is configured in the public example.

## What to adopt next

The repository keeps its control plane dependency-free while the first fixture
is built. The following projects are reference points or optional local
adapters, not bundled dependencies:

| Need | Candidate | Decision |
| --- | --- | --- |
| Deterministic browser control | [Playwright](https://github.com/microsoft/playwright) (Apache-2.0) | Use as the first web ERP/QMS adapter; selectors and accessibility data before vision. |
| Browser-agent fallback | [Browser Use](https://github.com/browser-use/browser-use) / [Browser Harness](https://github.com/browser-use/browser-harness) (MIT) | Optional fallback for a dedicated profile; keep domains, cookies, screenshots, and cloud use local and allowlisted. |
| Guided browser extraction | [Stagehand](https://github.com/browserbase/stagehand) (MIT) | Evaluate only if Playwright selectors cannot cover the fixture. |
| Durable scheduling | [Temporal](https://github.com/temporalio/temporal) (MIT) or [Trigger.dev](https://github.com/triggerdotdev/trigger.dev) | Do not replace SQLite until crash recovery, concurrency, or multi-machine scheduling is demonstrated as a need. |
| Office artifacts | [python-docx](https://github.com/python-openxml/python-docx), [openpyxl](https://github.com/ericgazoni/openpyxl), [python-pptx](https://github.com/scanny/python-pptx) | Use one renderer contract for DOCX/XLSX/PPTX and reopen each artifact before the manifest is accepted. |
| Agent traces | [Langfuse](https://github.com/langfuse/langfuse) (self-hostable, MIT core) | Add only after the local evidence manifest and redaction rules are stable. |
| Local model translation | [`Hskim-droid/local-llm`](https://github.com/Hskim-droid/local-llm) | Use its hardware/profile and local-engine ideas as a sibling integration; keep this harness's loopback translator and artifact contract independent. |

The synthetic proof exercises the same QMS records through DOCX, XLSX, and PPTX
one at a time. It measures record accuracy, evidence coverage, artifact
re-openability, duplicate suppression, and recovery after a forced stop before
any live ERP or mail permission is added.

## Run the synthetic browser proof

The fixture is the only adapter included in the public repository. It uses a
local HTML page, headless Chromium, and local office renderers; it contains no
business data or credentials.

```bash
python3 -m pip install -r requirements-fixture.txt
python3 -m playwright install chromium
python3 harness.py init
```

Set the local, ignored `config.json` value
`executor.extract` to `fixture-demo`, enqueue one QMS job with `--output-format
docx`, `xlsx`, or `pptx`, and run:

```bash
python3 fixture_demo.py \
  --config config.json \
  --job-id JOB_ID \
  --claim
```

The command reads `fixtures/qms_daily.html` through Chromium, validates three
records and their source references, creates exactly the requested office
artifact, reopens it and compares every record cell with the extracted view,
writes the manifest, and moves the specified queued job to `drafted`. It never
sends email or writes to an ERP/QMS system.

## Generic surface probe

`browser_probe.py` is the next adapter boundary. It receives a task contract
with semantic aliases rather than CSS selectors or a fixed menu path:

```bash
python3 browser_probe.py \
  --html fixtures/qms_variant_3.html \
  --task fixtures/qms_task.json
```

The probe first records the accessible surface and its capabilities, then uses
role/name based navigation, resolves table columns by aliases, and stops on an
ambiguous target. `table_terms` must identify the requested table through its
accessible label, field aliases are one-to-one, and `forbidden_terms` blocks
destructive-looking targets before any click. The three `qms_variant_*.html`
files deliberately change menu depth, language, column order, and DOM
structure while keeping the same task contract. This layer returns records and
observations; the queue and artifact contract remain separate so an unknown
app cannot silently become a write-capable connector. It is a browser
accessibility-tree adapter, not a universal native-app or visual/OCR adapter;
an app that exposes no reliable labels must stop or receive a dedicated
adapter.

`surface_adapter.py` fixes the cross-platform boundary. `UiObservation` carries
the normalized UI graph and capabilities, `ActionRequest` names an intended
action and must include the generation-specific `observation_id`, and
`ActionReceipt` records what the adapter accepted and executed against that
observation.
`PlaywrightAriaAdapter` is the first implementation; macOS AXUIElement,
Windows UI Automation, or Linux AT-SPI adapters can implement the same
`SurfaceAdapter` protocol at the control-plane boundary. The current QMS
planner still contains browser-specific table extraction, so native table
planning is a separate adapter task rather than an automatic drop-in.

`native_adapters.py` includes optional macOS AXUIElement and Windows UIA
implementations. They require an explicit application/window root and load
their platform binding only when used. Check the local capability report before
selecting one:

```bash
python3 -c 'from native_adapters import backend_status; import json; print(json.dumps([x.to_dict() for x in backend_status()], indent=2))'
```

Install only on the matching host (`pyobjc-framework-Quartz` on macOS,
`pywinauto` on Windows) and grant the operating system's accessibility
permission. A missing binding or missing explicit root fails closed; it never
falls back to desktop-wide clicking.

The native adapter tests use fake AX/UIA trees. Real host permission grants,
window discovery, and application-specific control patterns still require a
MacBook or Windows host validation pass.

`bootstrap.py` is a bounded capability mapper, not a general-purpose installer:
model weights, ERP connectors, mail credentials, and OS accessibility grants
remain explicit operator decisions. The local-llm sibling project can supply a
hardware-selected engine/model, but the public harness does not silently pull
large model files or assume that Ollama is installed.

The design lineage is recorded in
[`RPA_REFERENCE_LINEAGE.md`](RPA_REFERENCE_LINEAGE.md). It maps the RPA reviews,
Robot Framework/RPA Framework, TagUI, Playwright, macOS AX, Windows UIA,
BrowserGym, and OSWorld references to the code decisions and lists the gaps
that must be closed before enabling real ERP or mail permissions.
