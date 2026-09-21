#!/usr/bin/env python3
"""Cross-platform, draft-first queue for Codex-connected persona jobs.

The public repository only contains the control plane. UI extractors, document
renderers, and mail senders are local adapters configured outside git.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sqlite3
import sys
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


FORMATS = {"xlsx", "docx", "pptx"}
STATUSES = {"queued", "running", "drafted", "sent", "blocked", "failed"}
MANIFEST_VERSION = 1


def now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def repo_root() -> Path:
    return Path(__file__).resolve().parent


def config_path(value: str | None) -> Path:
    return Path(value).expanduser() if value else repo_root() / "config.json"


def load_config(path: Path) -> dict[str, Any]:
    if not path.exists():
        raise FileNotFoundError(
            f"missing config: {path}. Copy config.example.json to {path}"
        )
    with path.open(encoding="utf-8") as handle:
        config = json.load(handle)
    if config.get("mode") not in {"draft_only", "live"}:
        raise ValueError("config.mode must be draft_only or live")
    return config


def load_persona(persona_id: str) -> dict[str, Any]:
    """Load the checked-in policy declaration used to validate a job."""
    path = repo_root() / "personas" / f"{persona_id}.json"
    if not path.is_file():
        raise ValueError(f"unknown persona: {persona_id}")
    with path.open(encoding="utf-8") as handle:
        persona = json.load(handle)
    if persona.get("id") != persona_id:
        raise ValueError(f"persona id mismatch in {path}")
    if not persona.get("allowed_systems"):
        raise ValueError(f"persona has no allowed_systems: {persona_id}")
    if "draft_artifact" not in persona.get("allowed_actions", []):
        raise ValueError(f"persona cannot draft artifacts: {persona_id}")
    return persona


def path_from_config(config: dict[str, Any], key: str) -> Path:
    value = Path(str(config[key])).expanduser()
    return value if value.is_absolute() else repo_root() / value


def database_path(config: dict[str, Any]) -> Path:
    return path_from_config(config, "state_dir") / "queue.sqlite3"


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def safe_local_path(raw: str, root: Path) -> Path:
    """Resolve a local result path without allowing it to escape its store."""
    candidate = Path(raw).expanduser()
    if not candidate.is_absolute():
        candidate = root / candidate
    candidate = candidate.resolve()
    root = root.resolve()
    if candidate != root and root not in candidate.parents:
        raise ValueError(f"path must stay inside {root}")
    return candidate


class ManagedConnection(sqlite3.Connection):
    """Close SQLite connections when callers use the existing ``with`` style."""

    def __exit__(self, exc_type: Any, exc_value: Any, traceback: Any) -> None:
        try:
            super().__exit__(exc_type, exc_value, traceback)
        finally:
            self.close()


def validate_manifest(manifest: dict[str, Any], job: dict[str, Any], config: dict[str, Any]) -> tuple[Path, str]:
    """Validate adapter output before it can move a job to drafted."""
    if not isinstance(manifest, dict):
        raise ValueError("manifest must be a JSON object")
    required = {"schema_version", "job_id", "source_system", "captured_at", "records", "checks", "artifact"}
    missing = sorted(required - set(manifest))
    if missing:
        raise ValueError(f"manifest missing fields: {', '.join(missing)}")
    if manifest["schema_version"] != MANIFEST_VERSION:
        raise ValueError(f"unsupported manifest schema_version: {manifest['schema_version']}")
    if manifest["job_id"] != job["id"]:
        raise ValueError("manifest.job_id does not match the job")
    if manifest["source_system"] != job["source_system"]:
        raise ValueError("manifest.source_system does not match the job")
    if not isinstance(manifest["records"], list):
        raise ValueError("manifest.records must be a list")
    if not isinstance(manifest["checks"], list) or not manifest["checks"]:
        raise ValueError("manifest.checks must be a non-empty list")
    if any(not isinstance(check, dict) for check in manifest["checks"]):
        raise ValueError("manifest.checks entries must be objects")
    failed_checks = [check.get("name", "unnamed") for check in manifest["checks"] if check.get("passed") is not True]
    if failed_checks:
        raise ValueError(f"evidence checks failed: {', '.join(failed_checks)}")
    artifact = manifest["artifact"]
    if not isinstance(artifact, dict) or not {"path", "format", "sha256"}.issubset(artifact):
        raise ValueError("manifest.artifact requires path, format, and sha256")
    if artifact["format"] != job["output_format"]:
        raise ValueError("manifest.artifact.format does not match the job")
    artifact_path = safe_local_path(str(artifact["path"]), path_from_config(config, "artifact_dir"))
    if not artifact_path.is_file():
        raise ValueError(f"artifact does not exist: {artifact_path}")
    actual_hash = sha256_file(artifact_path)
    if actual_hash != artifact["sha256"]:
        raise ValueError("manifest.artifact.sha256 does not match the file")
    return artifact_path, actual_hash


def connect(config: dict[str, Any]) -> sqlite3.Connection:
    path = database_path(config)
    path.parent.mkdir(parents=True, exist_ok=True)
    connection = sqlite3.connect(path, factory=ManagedConnection)
    connection.row_factory = sqlite3.Row
    connection.execute("PRAGMA journal_mode=WAL")
    return connection


def init_schema(connection: sqlite3.Connection) -> None:
    connection.executescript(
        """
        CREATE TABLE IF NOT EXISTS jobs (
          id TEXT PRIMARY KEY,
          persona_id TEXT NOT NULL,
          source_system TEXT NOT NULL,
          request TEXT NOT NULL,
          output_format TEXT NOT NULL CHECK(output_format IN ('xlsx','docx','pptx')),
          recipients_json TEXT NOT NULL,
          trigger TEXT NOT NULL,
          idempotency_key TEXT,
          priority INTEGER NOT NULL DEFAULT 0,
          status TEXT NOT NULL DEFAULT 'queued',
          attempts INTEGER NOT NULL DEFAULT 0,
          artifact_path TEXT,
          artifact_sha256 TEXT,
          manifest_path TEXT,
          error TEXT,
          created_at TEXT NOT NULL,
          updated_at TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS events (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          job_id TEXT,
          event_type TEXT NOT NULL,
          detail_json TEXT NOT NULL,
          created_at TEXT NOT NULL
        );
        CREATE INDEX IF NOT EXISTS jobs_status_order
          ON jobs(status, priority DESC, created_at ASC);
        """
    )
    # The local runtime is intentionally migration-light. This keeps an ignored
    # SQLite file created by an earlier harness version usable after upgrades.
    columns = {row[1] for row in connection.execute("PRAGMA table_info(jobs)")}
    for name, definition in {
        "idempotency_key": "TEXT",
        "artifact_sha256": "TEXT",
        "manifest_path": "TEXT",
    }.items():
        if name not in columns:
            connection.execute(f"ALTER TABLE jobs ADD COLUMN {name} {definition}")
    connection.execute(
        "CREATE UNIQUE INDEX IF NOT EXISTS jobs_idempotency_key ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL"
    )
    connection.commit()


def event(connection: sqlite3.Connection, job_id: str | None, event_type: str, detail: Any) -> None:
    connection.execute(
        "INSERT INTO events(job_id,event_type,detail_json,created_at) VALUES (?,?,?,?)",
        (job_id, event_type, json.dumps(detail, ensure_ascii=False), now()),
    )
    connection.commit()


def ensure_config(path: Path) -> None:
    if path.exists():
        return
    example = path.parent / "config.example.json"
    if not example.exists():
        raise FileNotFoundError(f"missing example config: {example}")
    path.write_text(example.read_text(encoding="utf-8"), encoding="utf-8")


def cmd_init(args: argparse.Namespace) -> int:
    path = config_path(args.config)
    path.parent.mkdir(parents=True, exist_ok=True)
    ensure_config(path)
    config = load_config(path)
    path_from_config(config, "state_dir").mkdir(parents=True, exist_ok=True)
    path_from_config(config, "artifact_dir").mkdir(parents=True, exist_ok=True)
    with connect(config) as connection:
        init_schema(connection)
        event(connection, None, "HARNESS_INITIALIZED", {"config": str(path)})
    print(json.dumps({"ok": True, "config": str(path), "database": str(database_path(config))}, ensure_ascii=False))
    return 0


def cmd_doctor(args: argparse.Namespace) -> int:
    path = config_path(args.config)
    checks: dict[str, Any] = {"python": sys.version.split()[0], "config": str(path)}
    if not path.exists():
        checks["config_exists"] = False
        print(json.dumps(checks, ensure_ascii=False, indent=2))
        return 1
    try:
        config = load_config(path)
        checks.update(
            {
                "config_exists": True,
                "mode": config["mode"],
                "draft_only": config["mode"] == "draft_only",
                "state_dir": str(path_from_config(config, "state_dir")),
                "artifact_dir": str(path_from_config(config, "artifact_dir")),
                "executor_configured": any(config.get("executor", {}).values()),
            }
        )
        with connect(config) as connection:
            init_schema(connection)
        checks["database_ready"] = True
        checks["ok"] = True
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        checks["ok"] = False
        checks["error"] = str(exc)
    print(json.dumps(checks, ensure_ascii=False, indent=2))
    return 0 if checks.get("ok") else 1


def cmd_session_start(args: argparse.Namespace) -> int:
    path = config_path(args.config)
    config = load_config(path)
    with connect(config) as connection:
        init_schema(connection)
        event(
            connection,
            None,
            "SESSION_ATTACHED",
            {"agent": args.agent, "host": os.environ.get("COMPUTERNAME") or os.environ.get("HOSTNAME")},
        )
    print(json.dumps({"ok": True, "agent": args.agent, "mode": config["mode"]}, ensure_ascii=False))
    return 0


def cmd_enqueue(args: argparse.Namespace) -> int:
    if args.output_format not in FORMATS:
        raise ValueError(f"output format must be one of {sorted(FORMATS)}")
    path = config_path(args.config)
    config = load_config(path)
    persona = load_persona(args.persona)
    source_system = args.source_system.strip().upper()
    allowed_systems = {str(item).upper() for item in persona["allowed_systems"]}
    if source_system not in allowed_systems:
        raise ValueError(f"persona {args.persona} cannot read source system {source_system}")
    recipients = [item.strip() for item in args.recipient if item.strip()]
    allowlist = set(config.get("recipient_allowlist", []))
    if allowlist and any(item not in allowlist for item in recipients):
        raise ValueError("recipient is outside config.recipient_allowlist")
    idempotency_key = args.idempotency_key or hashlib.sha256(
        json.dumps(
            {
                "persona": args.persona,
                "source_system": source_system,
                "request": args.request,
                "output_format": args.output_format,
                "recipients": recipients,
                "trigger": args.trigger,
            },
            ensure_ascii=False,
            sort_keys=True,
        ).encode("utf-8")
    ).hexdigest()
    job_id = f"job-{datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')}-{uuid.uuid4().hex[:8]}"
    created = now()
    with connect(config) as connection:
        init_schema(connection)
        try:
            connection.execute(
                """INSERT INTO jobs
                (id,persona_id,source_system,request,output_format,recipients_json,trigger,idempotency_key,priority,status,created_at,updated_at)
                VALUES (?,?,?,?,?,?,?,?,?,?,?,?)""",
                (
                    job_id,
                    args.persona,
                    source_system,
                    args.request,
                    args.output_format,
                    json.dumps(recipients, ensure_ascii=False),
                    args.trigger,
                    idempotency_key,
                    args.priority,
                    "queued",
                    created,
                    created,
                ),
            )
        except sqlite3.IntegrityError:
            existing = connection.execute("SELECT id, status FROM jobs WHERE idempotency_key=?", (idempotency_key,)).fetchone()
            connection.rollback()
            print(json.dumps({"ok": True, "duplicate": True, "job_id": existing["id"], "status": existing["status"]}, ensure_ascii=False))
            return 0
        event(connection, job_id, "JOB_QUEUED", {"request": args.request, "format": args.output_format, "idempotency_key": idempotency_key})
    print(json.dumps({"ok": True, "job_id": job_id, "status": "queued"}, ensure_ascii=False))
    return 0


def row_json(row: sqlite3.Row) -> dict[str, Any]:
    value = dict(row)
    value["recipients"] = json.loads(value.pop("recipients_json"))
    return value


def cmd_status(args: argparse.Namespace) -> int:
    config = load_config(config_path(args.config))
    with connect(config) as connection:
        init_schema(connection)
        if args.job_id:
            row = connection.execute("SELECT * FROM jobs WHERE id=?", (args.job_id,)).fetchone()
            rows = [] if row is None else [row]
        else:
            rows = connection.execute(
                "SELECT * FROM jobs ORDER BY created_at DESC LIMIT ?", (args.limit,)
            ).fetchall()
    print(json.dumps([row_json(row) for row in rows], ensure_ascii=False, indent=2))
    return 0


def cmd_run_once(args: argparse.Namespace) -> int:
    config = load_config(config_path(args.config))
    with connect(config) as connection:
        init_schema(connection)
        connection.execute("BEGIN IMMEDIATE")
        if args.job_id:
            row = connection.execute(
                "SELECT * FROM jobs WHERE status='queued' AND id=? LIMIT 1", (args.job_id,)
            ).fetchone()
        else:
            row = connection.execute(
                "SELECT * FROM jobs WHERE status='queued' ORDER BY priority DESC, created_at ASC LIMIT 1"
            ).fetchone()
        if row is None:
            connection.commit()
            print(json.dumps({"ok": True, "message": "queue empty"}, ensure_ascii=False))
            return 0
        job = row_json(row)
        if args.dry_run:
            event(connection, job["id"], "DRY_RUN_PREVIEW", {"job": job})
            connection.commit()
            print(json.dumps({"ok": True, "dry_run": True, "job": job}, ensure_ascii=False, indent=2))
            return 0
        connection.execute(
            "UPDATE jobs SET status='running', attempts=attempts+1, updated_at=? WHERE id=?",
            (now(), job["id"]),
        )
        extractor = config.get("executor", {}).get("extract")
        if not extractor:
            error = "no local UI extractor configured; job left blocked in draft-only control plane"
            connection.execute(
                "UPDATE jobs SET status='blocked', error=?, updated_at=? WHERE id=?",
                (error, now(), job["id"]),
            )
            event(connection, job["id"], "JOB_BLOCKED", {"reason": error})
            connection.commit()
            print(json.dumps({"ok": False, "job_id": job["id"], "status": "blocked", "error": error}, ensure_ascii=False))
            return 2
        if not args.claim:
            error = "configured adapter was not claimed; rerun with --claim because the public harness never executes commands"
            connection.execute(
                "UPDATE jobs SET status='blocked', error=?, updated_at=? WHERE id=?",
                (error, now(), job["id"]),
            )
            event(connection, job["id"], "JOB_BLOCKED", {"reason": error, "executor": extractor})
            connection.commit()
            print(json.dumps({"ok": False, "job_id": job["id"], "status": "blocked", "error": error}, ensure_ascii=False))
            return 2
        event(connection, job["id"], "EXTRACTOR_HANDOFF_READY", {"executor": extractor})
        connection.commit()
        print(json.dumps({"ok": True, "completed": False, "handoff_required": True, "job_id": job["id"], "status": "running", "next": "local extractor adapter must produce a manifest and call record-result"}, ensure_ascii=False))
    return 0


def cmd_recover(args: argparse.Namespace) -> int:
    """Requeue jobs left running after a crashed local worker."""
    if args.age_seconds < 1:
        raise ValueError("age-seconds must be positive")
    config = load_config(config_path(args.config))
    cutoff = datetime.now(timezone.utc).timestamp() - args.age_seconds
    recovered: list[str] = []
    with connect(config) as connection:
        init_schema(connection)
        rows = connection.execute("SELECT id, updated_at FROM jobs WHERE status='running'").fetchall()
        for row in rows:
            try:
                updated = datetime.fromisoformat(row["updated_at"]).timestamp()
            except ValueError:
                continue
            if updated >= cutoff:
                continue
            connection.execute(
                "UPDATE jobs SET status='queued', error=?, updated_at=? WHERE id=?",
                (f"requeued after stale running lease ({args.age_seconds}s)", now(), row["id"]),
            )
            event(connection, row["id"], "JOB_REQUEUED", {"age_seconds": args.age_seconds})
            recovered.append(row["id"])
    print(json.dumps({"ok": True, "recovered": recovered}, ensure_ascii=False))
    return 0


def cmd_record_result(args: argparse.Namespace) -> int:
    config = load_config(config_path(args.config))
    manifest_path = Path(args.manifest).expanduser().resolve()
    if not manifest_path.is_file():
        raise FileNotFoundError(f"missing manifest: {manifest_path}")
    with manifest_path.open(encoding="utf-8") as handle:
        manifest = json.load(handle)
    with connect(config) as connection:
        init_schema(connection)
        row = connection.execute("SELECT * FROM jobs WHERE id=?", (args.job_id,)).fetchone()
        if row is None:
            raise ValueError(f"unknown job: {args.job_id}")
        job = row_json(row)
        if job["status"] != "running":
            raise ValueError(f"job must be running before recording a result (current: {job['status']})")
        artifact_path, artifact_hash = validate_manifest(manifest, job, config)
        connection.execute(
            "UPDATE jobs SET status='drafted', artifact_path=?, artifact_sha256=?, manifest_path=?, error=NULL, updated_at=? WHERE id=?",
            (str(artifact_path), artifact_hash, str(manifest_path), now(), args.job_id),
        )
        event(
            connection,
            args.job_id,
            "ARTIFACT_DRAFTED",
            {"artifact": str(artifact_path), "sha256": artifact_hash, "manifest": str(manifest_path), "record_count": len(manifest["records"])},
        )
    print(json.dumps({"ok": True, "job_id": args.job_id, "status": "drafted", "artifact": str(artifact_path), "sha256": artifact_hash}, ensure_ascii=False))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Draft-first Codex persona job harness")
    parser.add_argument("--config", help="path to local config.json")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("init", help="create local config/runtime state")
    sub.add_parser("doctor", help="check local runtime and safety defaults")

    session = sub.add_parser("session-start", help="record a Codex or local agent session")
    session.add_argument("--agent", default="codex")

    enqueue = sub.add_parser("enqueue", help="add one job to the FIFO/priority queue")
    enqueue.add_argument("--persona", default="han-gyeol")
    enqueue.add_argument("--source-system", required=True)
    enqueue.add_argument("--request", required=True)
    enqueue.add_argument("--output-format", choices=sorted(FORMATS), default="docx")
    enqueue.add_argument("--recipient", action="append", default=[])
    enqueue.add_argument("--trigger", choices=["scheduled", "email"], default="scheduled")
    enqueue.add_argument("--priority", type=int, default=0)
    enqueue.add_argument("--idempotency-key", help="stable key supplied by a scheduler or email adapter")

    status = sub.add_parser("status", help="show recent jobs")
    status.add_argument("--job-id")
    status.add_argument("--limit", type=int, default=20)

    run = sub.add_parser("run-once", help="take the next queued job")
    run.add_argument("--dry-run", action="store_true", help="preview without changing job state")
    run.add_argument("--claim", action="store_true", help="claim for an already-running local adapter; never executes a command")
    run.add_argument("--job-id", help="claim this exact queued job instead of the queue head")

    result = sub.add_parser("record-result", help="validate an adapter manifest and mark a job drafted")
    result.add_argument("--job-id", required=True)
    result.add_argument("--manifest", required=True, help="local JSON evidence manifest")

    recover = sub.add_parser("recover", help="requeue stale running jobs after a worker crash")
    recover.add_argument("--age-seconds", type=int, default=900)
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        return {
            "init": cmd_init,
            "doctor": cmd_doctor,
            "session-start": cmd_session_start,
            "enqueue": cmd_enqueue,
            "status": cmd_status,
            "run-once": cmd_run_once,
            "record-result": cmd_record_result,
            "recover": cmd_recover,
        }[args.command](args)
    except (FileNotFoundError, ValueError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
