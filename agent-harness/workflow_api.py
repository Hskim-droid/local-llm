"""Thin SDK for evidence-linked, draft-first local document workflows.

Adapters remain responsible for knowing how to read a particular file, browser,
native application, or local business system. This module gives those adapters
one stable handoff: collect normalized records, optionally translate them, and
render exactly one verified office artifact.
"""

from __future__ import annotations

from copy import deepcopy
from dataclasses import dataclass
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
from typing import Any, Callable, Mapping, Protocol, Sequence

from document_renderers import reopen_artifact, render_artifact
from workflow_contract import WorkflowRequest


class WorkflowError(RuntimeError):
    """The request or an adapter result cannot pass the workflow boundary."""


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


@dataclass(frozen=True)
class SourceBatch:
    """Normalized records returned by one local source adapter."""

    source_system: str
    captured_at: str
    columns: tuple[str, ...]
    records: tuple[Mapping[str, Any], ...]
    checks: tuple[Mapping[str, Any], ...] = ()

    def __post_init__(self) -> None:
        if not self.source_system.strip():
            raise ValueError("source_system must be non-empty")
        if not self.captured_at.strip():
            raise ValueError("captured_at must be non-empty")
        if not self.columns or len(set(self.columns)) != len(self.columns):
            raise ValueError("columns must contain unique names")
        records = tuple(self.records)
        object.__setattr__(self, "records", records)
        if any(not isinstance(record, Mapping) for record in records):
            raise ValueError("records must contain objects")
        checks = tuple(self.checks)
        object.__setattr__(self, "checks", checks)
        if any(not isinstance(check, Mapping) for check in checks):
            raise ValueError("checks must contain objects")


class SourceAdapter(Protocol):
    """Adapter boundary for files, media, browsers, native apps, or local ERP."""

    def collect(self, request: WorkflowRequest) -> SourceBatch:
        ...


RecordTranslator = Callable[
    [Sequence[Mapping[str, Any]], WorkflowRequest],
    Sequence[Mapping[str, Any]],
]


@dataclass(frozen=True)
class DraftArtifact:
    """Verified output returned to a coding tool or local caller."""

    artifact_path: Path
    manifest_path: Path
    artifact_format: str
    sha256: str
    record_count: int
    next_action: str = "human_review"

    def to_dict(self) -> dict[str, Any]:
        return {
            "artifact": {
                "path": str(self.artifact_path),
                "format": self.artifact_format,
                "sha256": self.sha256,
            },
            "manifest_path": str(self.manifest_path),
            "record_count": self.record_count,
            "next": self.next_action,
        }


def _passed(check: Mapping[str, Any]) -> bool:
    return check.get("passed") is True


def _validate_records(columns: Sequence[str], records: Sequence[Mapping[str, Any]]) -> list[dict[str, Any]]:
    if not records:
        raise WorkflowError("source adapter returned no records")
    normalized: list[dict[str, Any]] = []
    record_ids: set[str] = set()
    for index, record in enumerate(records, 1):
        item = dict(record)
        record_id = str(item.get("record_id", "")).strip()
        source_ref = str(item.get("source_ref", "")).strip()
        if not record_id:
            raise WorkflowError(f"record {index} is missing record_id")
        if not source_ref:
            raise WorkflowError(f"record {record_id} is missing source_ref")
        if record_id in record_ids:
            raise WorkflowError(f"duplicate record_id: {record_id}")
        missing = [column for column in columns if column not in item]
        if missing:
            raise WorkflowError(f"record {record_id} is missing columns: {missing}")
        record_ids.add(record_id)
        normalized.append(item)
    return normalized


def run_local_document_workflow(
    request: WorkflowRequest,
    adapter: SourceAdapter,
    output_path: str | Path,
    translator: RecordTranslator | None = None,
) -> DraftArtifact:
    """Collect, verify, optionally translate, and render one local draft.

    The adapter is the only component that knows how to inspect a source. The
    SDK never clicks, logs in, sends mail, or writes back to a remote surface.
    It keeps the original records in the manifest even when a translator returns
    a derived view for the document.
    """

    if not isinstance(request, WorkflowRequest):
        raise TypeError("request must be a WorkflowRequest")
    batch = adapter.collect(request)
    if not isinstance(batch, SourceBatch):
        raise WorkflowError("source adapter must return SourceBatch")
    checks = [dict(check) for check in batch.checks]
    failed = [str(check.get("name", "unnamed")) for check in checks if not _passed(check)]
    if failed:
        raise WorkflowError(f"source checks failed: {', '.join(failed)}")
    original_records = _validate_records(batch.columns, batch.records)
    document_records = deepcopy(original_records)
    if translator is not None:
        translated = translator(original_records, request)
        document_records = _validate_records(batch.columns, translated)
        if [row["record_id"] for row in document_records] != [row["record_id"] for row in original_records]:
            raise WorkflowError("translator changed record identity or ordering")
        translation_check = {"name": "translation_preserves_source_refs", "passed": all(
            row["source_ref"] == original["source_ref"]
            for row, original in zip(document_records, original_records)
        )}
        checks.append(translation_check)
        if not translation_check["passed"]:
            raise WorkflowError("translator changed source references")
    output = Path(output_path).expanduser()
    if output.suffix.lower() != f".{request.output_format}":
        raise WorkflowError(f"output path must use .{request.output_format}")
    output.parent.mkdir(parents=True, exist_ok=True)
    render_artifact(output, request.output_format, batch.source_system, batch.captured_at, batch.columns, document_records)
    reopen_check = reopen_artifact(output, request.output_format, batch.columns, document_records)
    if reopen_check.get("passed") is not True:
        raise WorkflowError("rendered artifact failed its reopen check")
    checks.append(dict(reopen_check))
    artifact_hash = sha256_file(output)
    manifest_path = output.with_suffix(".manifest.json")
    manifest = {
        "workflow_schema_version": 1,
        "request": request.to_dict(),
        "source_system": batch.source_system,
        "captured_at": batch.captured_at,
        "records": original_records,
        "document_records": document_records,
        "checks": checks,
        "artifact": {
            "path": str(output),
            "format": request.output_format,
            "sha256": artifact_hash,
        },
        "created_at": utc_now(),
        "next": "human_review",
    }
    manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2), encoding="utf-8")
    return DraftArtifact(output, manifest_path, request.output_format, artifact_hash, len(original_records))
