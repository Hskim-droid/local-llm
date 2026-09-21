from __future__ import annotations

import json
from pathlib import Path
import tempfile
import unittest
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from workflow_api import SourceBatch, WorkflowError, run_local_document_workflow
from workflow_contract import InputRef, WorkflowRequest


class FixtureAdapter:
    def collect(self, request: WorkflowRequest) -> SourceBatch:
        return SourceBatch(
            source_system="fixture",
            captured_at="2026-09-21T00:00:00+00:00",
            columns=("record_id", "title", "source_ref"),
            records=({"record_id": "A-1", "title": "원문", "source_ref": "fixture://A-1"},),
            checks=({"name": "fixture_freshness", "passed": True},),
        )


class WorkflowContractTests(unittest.TestCase):
    def request(self, output_format: str = "docx") -> WorkflowRequest:
        return WorkflowRequest(
            task="정리",
            inputs=(InputRef("file", "/tmp/source.txt", media_type="text/plain"),),
            output_format=output_format,
            target_language="en",
        )

    def test_json_nulls_are_rejected(self) -> None:
        with self.assertRaises(ValueError):
            InputRef.from_mapping({"kind": "file", "locator": None})
        with self.assertRaises(ValueError):
            WorkflowRequest.from_mapping({
                "task": None,
                "inputs": [{"kind": "file", "locator": "/tmp/source.txt"}],
                "output": {"format": "docx"},
            })

    def test_browser_input_requires_allowlist(self) -> None:
        with self.assertRaises(ValueError):
            InputRef("browser", "private-handle")

    def test_request_serializes_structured_inputs(self) -> None:
        payload = self.request("xlsx").to_dict()
        self.assertEqual(payload["output"]["format"], "xlsx")
        self.assertEqual(payload["inputs"][0]["kind"], "file")
        self.assertEqual(payload["policy"]["mode"], "draft_only")
        restored = WorkflowRequest.from_mapping(payload)
        self.assertEqual(restored, self.request("xlsx"))

    def test_translation_request_requires_translator(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(WorkflowError):
                run_local_document_workflow(
                    self.request(), FixtureAdapter(), Path(directory) / "draft.docx"
                )

    def test_workflow_renders_and_writes_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            result = run_local_document_workflow(
                self.request(), FixtureAdapter(), Path(directory) / "draft.docx",
                translator=lambda records, _request: [
                    {**record, "title": "translated"} for record in records
                ],
            )
            self.assertTrue(result.artifact_path.is_file())
            manifest = json.loads(result.manifest_path.read_text(encoding="utf-8"))
            self.assertEqual(manifest["records"][0]["title"], "원문")
            self.assertEqual(manifest["document_records"][0]["title"], "translated")
            self.assertEqual(manifest["next"], "human_review")

    def test_workflow_protects_original_from_in_place_translator(self) -> None:
        def mutating_translator(records, _request):
            records[0]["source_ref"] = "untrusted://changed"
            records[0]["title"] = "changed"
            return records

        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(WorkflowError):
                run_local_document_workflow(
                    self.request(), FixtureAdapter(), Path(directory) / "draft.docx",
                    translator=mutating_translator,
                )

    def test_workflow_rejects_translator_source_reference_change(self) -> None:
        def unsafe_translator(records, _request):
            return [{**record, "source_ref": "untrusted://changed"} for record in records]

        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(WorkflowError):
                run_local_document_workflow(
                    self.request(), FixtureAdapter(), Path(directory) / "draft.docx",
                    translator=unsafe_translator,
                )

    def test_workflow_rejects_missing_source_reference(self) -> None:
        class BadAdapter(FixtureAdapter):
            def collect(self, request: WorkflowRequest) -> SourceBatch:
                batch = super().collect(request)
                return SourceBatch(batch.source_system, batch.captured_at, batch.columns, (
                    {"record_id": "A-1", "title": "원문", "source_ref": None},
                ), batch.checks)

        request = WorkflowRequest(
            task="정리",
            inputs=(InputRef("file", "/tmp/source.txt", media_type="text/plain"),),
            output_format="docx",
        )
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(WorkflowError, "missing source_ref"):
                run_local_document_workflow(request, BadAdapter(), Path(directory) / "draft.docx")
