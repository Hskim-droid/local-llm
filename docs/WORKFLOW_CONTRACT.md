# Local-first document workflow contract

`local-llm` is positioned as a **local-first, draft-first UI-to-document
automation framework**. A coding tool, a scheduled local process, or a human
operator supplies a task contract; a source adapter observes one or more local
inputs; the pipeline preserves evidence, optionally translates or transcribes
locally, and produces exactly one reviewable DOCX, XLSX, or PPTX draft.

The public code is a framework boundary, not a universal crawler and not a
production ERP or mail connector.

## Canonical function shape

The implemented thin SDK exposes this contract through
`agent-harness/workflow_contract.py` and `agent-harness/workflow_api.py`. Its
public function is equivalent to:

```text
run_local_document_workflow(
    inputs,
    task,
    output_format,
    source_language="auto",
    target_language=None,
    policy={"mode": "draft_only", "approval": "required"},
) -> DraftArtifact
```

A request is represented by data rather than by an unrestricted natural-language
instruction:

```json
{
  "inputs": [
    {
      "kind": "browser",
      "locator": "local adapter reference",
      "scope": {"allowed_domains": ["qms.example.local"]}
    },
    {
      "kind": "file",
      "locator": "/local/work/source.pdf",
      "media_type": "application/pdf"
    }
  ],
  "task": "요약하고 미처리 항목을 담당자별로 정리",
  "source_language": "auto",
  "target_language": "en",
  "output": {"format": "docx"},
  "policy": {"mode": "draft_only", "approval": "required"}
}
```

The result keeps the document and its evidence together:

```json
{
  "artifact": {"format": "docx", "path": "runtime/artifacts/draft.docx", "sha256": "..."},
  "facts": [
    {"value": "Q-123", "status": "confirmed", "source_ref": "qms://issue/Q-123"},
    {"value": "번역 결과", "status": "translated", "source_ref": "qms://issue/Q-123#title"}
  ],
  "checks": [{"name": "source_freshness", "passed": true}],
  "next": "human_review"
}
```

`source_ref` is required for extracted facts. A translation or summary does not
replace the original value. Values that cannot be tied to an input are marked
`NEEDS_CHECK` and remain visible for review; they are not silently invented.
The SDK does not infer confidentiality from a screen or file. A local adapter or
policy layer must classify sensitive records before they enter a manifest; the
public core never uploads runtime state, but it cannot decide business
confidentiality on its own.

## Input families

| Input family | Adapter boundary | Current public status |
| --- | --- | --- |
| Local text, PDF, Office files, and images | File extractor returns records and source references | The Go document engine supports local document jobs; unified contract wiring is the next integration step |
| Foreign-language documents | Extract → preserve original → local translation → render | Loopback translation and field-level evidence are implemented in the harness fixture |
| Audio and video recordings | Local transcription adapter returns timestamped text and media references | Local engine paths exist; a single cross-input adapter contract is not yet validated |
| Browser web app | Playwright accessibility observation, bounded aliases, allowlisted origin | Synthetic browser fixture validated; real ERP/QMS hosts remain adapter work |
| Native Mac/Windows application | AXUIElement or Windows UI Automation observation | Normalized adapter contracts and fake-tree tests exist; host permission validation remains |
| Local ERP/QMS or desktop program | Explicit, least-privileged read adapter | No production connector is bundled or enabled |
| Web crawling | Bounded, allowlisted read observation with evidence | Generic unrestricted crawling is intentionally outside the contract |

An input adapter may read, search, and collect. It may not write to a remote
surface, send mail, or upload source material unless a separate adapter,
allowlist, and human approval gate are explicitly configured.

## Pipeline stages

1. **Discover** — inspect the declared file, media, browser, or native surface and
   record its capability and capture time.
2. **Extract** — return normalized records, source references, timestamps, and
   evidence selectors or media offsets.
3. **Normalize** — assign stable record IDs, remove duplicate records, and hash
   the observation where possible.
4. **Translate or transcribe locally** — keep the original and derived text
   side-by-side. The Ollama-compatible path is loopback-only and does not provide
   a cloud fallback.
5. **Verify provenance** — run freshness, duplicate, completeness, language, and
   artifact checks. Unverified facts carry an explicit status.
6. **Render one artifact** — choose exactly one of `docx`, `xlsx`, or `pptx`, then
   reopen it and verify its headers and record cells.
7. **Draft and hand off** — store the artifact hash and manifest in the local
   queue. The default next step is human review; external delivery is separate.

The current implementation covers the queue, idempotency key, UI observation
contracts, fixture browser extraction, loopback translation, DOCX/XLSX/PPTX
renderers, artifact reopen checks, manifest hash validation, and the thin SDK
function described above.
The missing production pieces are deliberately adapters: real host permissions,
ERP/QMS connectors, transcription selection, mail providers, and unattended
scheduling.

## Coding-tool attachment

A coding tool can call the queue through the existing CLI or a future thin SDK;
it should pass structured input references and an explicit output format rather
than asking the harness to guess a screen or file. The local launcher initializes
state, but cloning the repository does not authorize code execution, browser
access, remote writes, or mail delivery.

A host-specific adapter belongs in local ignored configuration or a separate
private package. Public fixtures must stay synthetic, and runtime state,
cookies, source documents, model weights, and recipient lists must stay outside
the public repository.

## Globalization boundary

The framework becomes globally reusable by adding adapters and policy profiles,
not by adding universal selectors or a cloud agent. A new integration should
supply:

- a source kind and capability report;
- an allowlist and least-privileged account or local file scope;
- normalized records with stable IDs and `source_ref` values;
- locale and language mappings for extraction and translation;
- one renderer-compatible record schema;
- a fixture, provenance checks, and a documented human approval boundary.

That keeps the core contract stable across foreign-language documents, local
programs, web apps, and browser-accessible ERP screens while allowing each host
to keep its credentials and permissions private.
