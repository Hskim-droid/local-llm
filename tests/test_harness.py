import json
import hashlib
import tempfile
import unittest
from pathlib import Path

import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
import harness  # noqa: E402


class HarnessTests(unittest.TestCase):
    def write_config(self, temp_path, *, extractor=None):
        config_path = temp_path / "config.json"
        config_path.write_text(
            json.dumps(
                {
                    "persona_id": "han-gyeol",
                    "persona_name": "한결",
                    "mode": "draft_only",
                    "state_dir": str(temp_path / "state"),
                    "artifact_dir": str(temp_path / "artifacts"),
                    "default_output_format": "docx",
                    "recipient_allowlist": [],
                    "executor": {"extract": extractor, "render": None, "send": None},
                }
            ),
            encoding="utf-8",
        )
        (temp_path / "artifacts").mkdir()
        return config_path

    def test_enqueue_and_dry_run_preserve_queued_state(self):
        with tempfile.TemporaryDirectory() as temp:
            temp_path = Path(temp)
            config_path = self.write_config(temp_path)

            self.assertEqual(
                harness.main(
                    [
                        "--config",
                        str(config_path),
                        "enqueue",
                        "--source-system",
                        "QMS",
                        "--request",
                        "daily report",
                        "--output-format",
                        "docx",
                    ]
                ),
                0,
            )
            self.assertEqual(
                harness.main(["--config", str(config_path), "run-once", "--dry-run"]),
                0,
            )
            config = harness.load_config(config_path)
            with harness.connect(config) as connection:
                row = connection.execute("SELECT status FROM jobs").fetchone()
            self.assertEqual(row["status"], "queued")

    def test_duplicate_idempotency_key_does_not_create_a_second_job(self):
        with tempfile.TemporaryDirectory() as temp:
            temp_path = Path(temp)
            config_path = self.write_config(temp_path)
            args = [
                "--config",
                str(config_path),
                "enqueue",
                "--source-system",
                "QMS",
                "--request",
                "daily report 2026-09-21",
                "--idempotency-key",
                "mail-message-123",
            ]
            self.assertEqual(harness.main(args), 0)
            self.assertEqual(harness.main(args), 0)
            config = harness.load_config(config_path)
            with harness.connect(config) as connection:
                self.assertEqual(connection.execute("SELECT COUNT(*) FROM jobs").fetchone()[0], 1)

    def test_record_result_requires_passing_checks_and_matching_hash(self):
        with tempfile.TemporaryDirectory() as temp:
            temp_path = Path(temp)
            config_path = self.write_config(temp_path, extractor="local-qms-adapter")
            self.assertEqual(
                harness.main(
                    [
                        "--config",
                        str(config_path),
                        "enqueue",
                        "--source-system",
                        "QMS",
                        "--request",
                        "daily report 2026-09-21",
                    ]
                ),
                0,
            )
            config = harness.load_config(config_path)
            with harness.connect(config) as connection:
                job_id = connection.execute("SELECT id FROM jobs").fetchone()[0]
            self.assertEqual(harness.main(["--config", str(config_path), "run-once", "--claim"]), 0)

            artifact = temp_path / "artifacts" / "daily.docx"
            artifact.write_bytes(b"synthetic docx artifact")
            manifest = {
                "schema_version": 1,
                "job_id": job_id,
                "source_system": "QMS",
                "captured_at": "2026-09-21T09:00:00+00:00",
                "records": [{"record_id": "Q-123", "source_ref": "qms://issue/Q-123"}],
                "checks": [{"name": "freshness", "passed": True}],
                "artifact": {
                    "path": str(artifact),
                    "format": "docx",
                    "sha256": hashlib.sha256(artifact.read_bytes()).hexdigest(),
                },
            }
            manifest_path = temp_path / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            self.assertEqual(
                harness.main(
                    ["--config", str(config_path), "record-result", "--job-id", job_id, "--manifest", str(manifest_path)]
                ),
                0,
            )
            with harness.connect(config) as connection:
                row = connection.execute("SELECT status, artifact_sha256 FROM jobs WHERE id=?", (job_id,)).fetchone()
            self.assertEqual(row["status"], "drafted")
            self.assertEqual(row["artifact_sha256"], manifest["artifact"]["sha256"])

    def test_persona_rejects_unknown_source_system(self):
        with tempfile.TemporaryDirectory() as temp:
            config_path = self.write_config(Path(temp))
            self.assertEqual(
                harness.main(
                    [
                        "--config",
                        str(config_path),
                        "enqueue",
                        "--source-system",
                        "CRM",
                        "--request",
                        "x",
                    ]
                ),
                1,
            )

    def test_claim_job_id_does_not_take_queue_head(self):
        with tempfile.TemporaryDirectory() as temp:
            temp_path = Path(temp)
            config_path = self.write_config(temp_path, extractor="local-qms-adapter")
            for request in ("first", "second"):
                self.assertEqual(
                    harness.main(
                        [
                            "--config",
                            str(config_path),
                            "enqueue",
                            "--source-system",
                            "QMS",
                            "--request",
                            request,
                            "--idempotency-key",
                            request,
                        ]
                    ),
                    0,
                )
            config = harness.load_config(config_path)
            with harness.connect(config) as connection:
                target_id = connection.execute("SELECT id FROM jobs WHERE request='second'").fetchone()[0]
            self.assertEqual(
                harness.main(["--config", str(config_path), "run-once", "--claim", "--job-id", target_id]),
                0,
            )
            with harness.connect(config) as connection:
                statuses = dict(connection.execute("SELECT request, status FROM jobs").fetchall())
            self.assertEqual(statuses, {"first": "queued", "second": "running"})

    def test_invalid_output_format_is_rejected(self):
        with self.assertRaises(SystemExit):
            harness.build_parser().parse_args(
                [
                    "enqueue",
                    "--source-system",
                    "QMS",
                    "--request",
                    "x",
                    "--output-format",
                    "pdf",
                ]
            )


if __name__ == "__main__":
    unittest.main()
