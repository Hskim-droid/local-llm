# Publication strategy for local-first document automation

Status: canonical working strategy, reviewed 2026-09-22

This document defines how `local-llm` and `dropkit.contents` publish work under
the positioning:

> **Local-first, draft-first UI-to-document automation framework**

The strategy is evidence-led. A public page may explain an idea before it is
implemented, but an implemented result is published only with a clear status,
an exact source revision, and a reviewable evidence packet.

## Language policy

All public-facing content is English-only: research pages, case notes,
technical publication notes, evidence-packet narratives, README-facing project
copy, LinkedIn profile copy, and LinkedIn posts. Product localization and
runtime translation capabilities are separate implementation features; they do
not change the language of the public research and showcase layer.

## 1. Decision

We publish a **trust chain**, not a feature stream:

```text
claim → bounded implementation → pinned run → artifact + manifest → narrative page → social pointer
```

The post is the last layer. It is not the source of truth, the test report, or
the release channel.

The canonical public unit is an **evidence packet**:

1. one bounded question or claim;
2. one pinned source revision or release;
3. synthetic or explicitly permitted inputs;
4. one or more output artifacts, or an explicit failed output;
5. a machine-readable verification record with hashes and checks;
6. a reproduction command and environment scope;
7. limitations, untested paths, and the next decision.

This follows the strongest common pattern in the references: BrowserGym
separates setup, usage, demo, benchmarks, paper, and citation; OSWorld-V2 pins
the complete evaluation release and keeps per-run provenance separate; GitHub
defines Releases as tagged, shareable software iterations. See
[`docs/research/PUBLICATION_REFERENCE_CAPTURE_20260922.md`](research/PUBLICATION_REFERENCE_CAPTURE_20260922.md).

## 2. Repository ownership

| Publication object | Canonical home | What the other repository may do |
| --- | --- | --- |
| Go engine, Python harness, SDK, schemas, tests, synthetic fixtures | `Hskim-droid/local-llm` | Link to an exact path, tag, or commit |
| Installable Windows/macOS package | GitHub Release in `local-llm` | Explain who should download it and link to the release |
| Workflow contract and reference lineage | `local-llm/docs/` | Summarize the contract; do not fork it into a second technical truth |
| Evidence packet and verification manifest | `local-llm/examples/<packet-id>/` is the single canonical packet location; large artifacts may be attached to a Release but must be referenced by the packet manifest | Render a readable case note and link to the exact packet files; generated copies must carry the packet ID and revision |
| Project introduction, case note, research note, social copy | `dropkit.contents` | Link to code, evidence, and release assets; do not recreate hashes or test results |
| Private adapter, credentials, real ERP/QMS data, recipient lists, local runtime state | Private/local configuration only | Never publish |

The existing split is correct. `local-llm` is the technical source and
`dropkit.contents` is the publication shell. The shell must remain a view of the
technical source, not a second implementation repository.

## 3. Publication classes

Every item gets exactly one primary class.

| Class | Use when | Required evidence | Primary channel |
| --- | --- | --- | --- |
| `release` | A user can download and run a versioned engine or pack | tag, CI result, install check, asset checksum, known limits | GitHub Release |
| `evidence_packet` | A bounded workflow or experiment has a reviewable result | pinned revision, fixture/input identity, output, verification JSON, checklist, limitations | `local-llm` packet + hub case page |
| `contract_note` | A schema/API/adapter boundary is stable enough for reuse | source path, examples, tests, compatibility notes, explicit non-goals | `local-llm/docs/` + project page |
| `research_note` | External references change a design decision | source ledger, claim status, comparison, decision, unresolved question | `dropkit.contents` reading + linked technical note |
| `social_pointer` | A durable item already exists elsewhere | one sentence, status, canonical URL, no new unsupported claim | LinkedIn/X/etc. |

An experiment without a reproducible input and verification record is a
`research_note` or `draft`, not an `evidence_packet`. A screenshot alone is
never a release or a benchmark. The existing scan-recovery page is a useful
predecessor, not yet a complete packet under this contract: its record has an
engine hash and output hash, but not every input hash, a source revision, or an
exact rerun command, and its checklist is still a template.

## 4. Evidence status vocabulary

Use one status in the title area and manifest. Do not use “works”, “production
ready”, “global”, or “supports” without a status and scope.

| Status | Meaning | Allowed wording |
| --- | --- | --- |
| `documented` | Described in code/docs; not independently rerun for this publication | “The contract defines…” |
| `fixture_verified` | Execution passed against the named synthetic fixture and declared checks | “The fixture execution passed…” |
| `locally_reproduced` | Execution was rerun on the named local environment with a saved record | “This local execution produced…” |
| `host_verified` | Execution passed on the named Mac/Windows/app surface | “Execution passed on [host scope]…” |

Execution status is separate from review and acceptance:

| Field | Values | Meaning |
| --- | --- | --- |
| `execution_status` | `not_run`, `passed`, `failed`, `partial` | What the declared command and checks actually established |
| `review_status` | `not_run`, `pending`, `passed`, `failed` | Whether a person reviewed the specified artifact/version/scope |
| `acceptance_status` | `not_recorded`, `accepted`, `rejected` | Whether the intended user explicitly accepted the result |

`fixture_verified` never implies `review_status=passed`. A runtime value of
`next: human_review` must remain `review_status=pending` until that review is
recorded. A failed or partial execution can be published as a failure note if
the failure itself is the bounded result; it must not be rewritten as a pass.

The current public harness should normally use `documented` or
`fixture_verified`. The scan-recovery case is a useful predecessor because it
exposes synthetic inputs, output, verification JSON, a checklist template,
an exact mismatch, and untested limits; it is not yet a complete
`fixture_verified` packet under the contract below.

## 5. Evidence packet contract

The packet ID is stable and human-readable, for example
`scan-recovery-20260915` or `qms-fixture-docx-20260922`.

Minimum packet layout:

```text
examples/<packet-id>/
  README.md                 # question, scope, result, limits, rerun command
  verification.json         # machine-readable checks and hashes
  review-checklist.csv      # human review fields, if visual/content review matters
  preview.png               # optional, synthetic or redacted only
  artifact.<ext>            # optional when small; otherwise release/object link
```

The publication manifest is deliberately separate from the runtime job
manifest. It should contain at least:

```json
{
  "schema_version": 1,
  "packet_id": "qms-fixture-docx-20260922",
  "claim": "A declared browser fixture can produce one DOCX draft with a passing reopen check",
  "status": "fixture_verified",
  "execution_status": "passed",
  "review_status": "pending",
  "acceptance_status": "not_recorded",
  "source": {
    "repository": "Hskim-droid/local-llm",
    "revision": "<full immutable commit SHA>",
    "workflow": "<CI run URL or local command record>"
  },
  "inputs": [{
    "id": "fixture-1",
    "kind": "synthetic",
    "path_or_generator": "examples/qms-fixture/fixture.html",
    "sha256": "..."
  }],
  "outputs": [{"path": "artifact.docx", "sha256": "..."}],
  "command": "python3 ... --fixture examples/qms-fixture/fixture.html",
  "configuration": "examples/qms-fixture/config.json",
  "dependencies": ["playwright==<version>", "python-docx==<version>"],
  "checks": [{
    "name": "artifact_reopen",
    "passed": true,
    "establishes": "document structure and declared record cells"
  }],
  "environment": {"os": "<scope>", "runtime": "<versions>"},
  "limitations": ["real ERP/QMS host not tested"],
  "next": "human_review"
}
```

The manifest is a **publication projection**, not a copy of the runtime job
manifest. Runtime manifests may retain original records and request data for
local validation; they must be redacted and reduced before publication. The
publication review covers the JSON, document metadata, previews, embedded
images, and downloadable artifact. It must never contain cookies, credentials,
source business records, recipient lists, or a host username/path that
identifies the machine. A hash of a private input does not make that input
anonymous or publicly reproducible.

## 6. Release and link rules

### Installable software

1. Build from a pinned commit and record its full SHA.
2. Run the matching Go/Python/installer checks.
3. Publish a Git tag and GitHub Release with notes, assets, checksums, and known
   limitations.
4. Link users to `/releases/latest` or the direct asset URL only for the stable
   install lane.
5. If the release changes security behavior, publish the security information
   with it.

GitHub Releases are for deployable software iterations; they are not a place to
store an unverified narrative or every development checkpoint.

### Framework and harness changes

Until the harness has its own stable install and compatibility policy, publish
it as a dated contract/evidence checkpoint tied to an immutable commit. Do not
call the current `main` branch a product release.

As of this strategy review, `v0.9.4` is the latest public binary release and
predates the integrated harness/SDK commits on `main`. The README must keep the
Windows download path and the experimental harness path visibly separate.

### Evidence pages

The hub page should link in this order:

1. readable result page;
2. exact evidence packet;
3. exact source revision or release;
4. reproduction command;
5. limitations and untested scope.

The page may use a preview image, but the image is navigation and explanation,
not proof. The downloadable verification record is the proof-bearing object.
The page must say what each check establishes and what it does not establish:

- an input/output hash establishes artifact identity, not content accuracy;
- an artifact reopen check establishes limited structural validity, not
  translation accuracy or visual fidelity;
- a semantic comparison establishes only the declared fields and fixture;
- a privacy review establishes only the reviewed public packet;
- an elapsed-time result is not a general performance benchmark.

“Repeatable” means the documented command can be rerun against the same
declared inputs. “Reproducible build” is a stronger byte-identical claim and
must not be used unless the environment and comparison protocol support it.

### Social posts

Social copy is a pointer, not a new publication layer. It must contain:

- the status label;
- the one-sentence result;
- the canonical page URL;
- no number, speed claim, quality claim, or “production” claim absent from the
  evidence packet.

## 7. Publication sequence

```text
1. Define one bounded claim and non-claims
2. Freeze fixture/input identity and source revision
3. Run the smallest useful check
4. Write artifact + manifest + limitations
5. Run tests/CI, inspect the actual output, and map each check to the claim
6. Publish the technical packet or release
7. Publish the human-readable hub page
8. Send social pointers to the hub page
9. Record corrections as a new revision, never by silently editing history
```

The “smallest useful check” is important. For a translation workflow it may be
one fixed multilingual fixture with original and translated fields preserved;
for UI extraction it may be one synthetic browser variant and one artifact
reopen check. Real ERP, mail, host permissions, and unattended sending are
separate gates, not implied by a synthetic pass.

## 8. What is already strong and what is missing

### Already reflected

- `local-llm/README.md` distinguishes documented features from independently
  reproduced results and points users to the correct engine/harness boundaries.
- `docs/WORKFLOW_CONTRACT.md` requires source references, one explicit output,
  manifest integrity, and human review as the next step.
- `docs/RPA_REFERENCE_LINEAGE.md` records reference-to-design decisions and
  explicitly lists unimplemented real-host, mail, and scheduler paths.
- The existing `scan-recovery` public case already exposes a synthetic input,
  preview, output, verification JSON, review checklist template, and
  limitations. It is a useful predecessor, but does not yet satisfy the full
  packet contract above.

### Missing or inconsistent

1. There is no shared publication manifest schema or packet ID convention.
2. The current engine release lane and the new harness/SDK lane are not yet
   represented by separate versioned release/checkpoint labels.
3. The hub project page still describes a dated snapshot and does not yet link
   to a standard evidence-packet index.
4. `local-llm` has no `CITATION.cff`; adding one is useful when the reusable
   framework reaches a stable checkpoint, but is not required for the current
   experimental lane.
5. Binary releases do not yet have a CI-built attestation lane. This is a later
   supply-chain improvement, not a prerequisite for synthetic case notes.

## 9. Implementation order

### P1 — publication contract

- Add a shared `publication-manifest` example and validator in `local-llm`.
- Convert the next synthetic workflow into the packet layout above.
- Add a hub index that links packet ID, status, source revision, and limits.

Success: one new packet can be checked locally and from a clean clone without
reading a chat transcript; validation rejects a changed artifact and a
publication manifest containing a private path.

Stop/redirect: if one manifest cannot represent both a document conversion and
a UI-to-document run without optional-field confusion, split the schemas into
`evidence-manifest` and `release-manifest` rather than making a permissive blob.

### P2 — release clarity

- Add a concise changelog or release-lane page.
- Label the current binary release as engine-only and keep harness checkpoints
  tied to exact commits until compatibility is stable.
- Add `CITATION.cff` at the first stable reusable framework checkpoint.

Success: a new visitor can tell within one minute which link installs software,
which link demonstrates an experiment, and which link describes a contract.

### P3 — stronger distribution provenance

- Build release assets in CI and publish checksums.
- Consider GitHub artifact attestations for binaries and packaged manifests.
- Consider Zenodo/DOI only for a durable framework or research release, not for
  every development note.

Success: an external user can identify the source revision and build path for a
downloaded binary without trusting a prose assertion alone.

## 10. Non-goals

- Do not publish real ERP/QMS screens, cookies, credentials, mail addresses, or
  customer documents.
- Do not present a synthetic fixture as production validation.
- Do not make the publication hub the technical source of truth.
- Do not add a cloud model or remote sender merely to make a demo easier to
  publish.
- Do not use a floating `main`/`latest` link when claiming that a run is
  reproducible.

## References

The dated source capture and internal mapping are kept in
[`docs/research/PUBLICATION_REFERENCE_CAPTURE_20260922.md`](research/PUBLICATION_REFERENCE_CAPTURE_20260922.md).
The external sources include GitHub repository/release/citation guidance,
BrowserGym, OSWorld-V2, Reproducible Builds, MarkItDown, OpenHands, and
Paperless-ngx. External pages were reviewed on 2026-09-22; their current
content may change.
