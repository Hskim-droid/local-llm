import json
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch
from urllib.request import ProxyHandler

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from translation_pipeline import (  # noqa: E402
    OllamaTranslator,
    PassthroughTranslator,
    TranslationError,
    TranslationRequest,
    translate_records,
)
import translation_pipeline  # noqa: E402


class _FakeResponse:
    def __init__(self, payload):
        self.payload = payload

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def read(self, _size):
        return json.dumps(self.payload).encode("utf-8")


class _RemoteTranslator:
    backend_name = "remote"
    model_name = "remote"
    local_only = False

    def translate_text(self, text, request):
        return text


class TranslationPipelineTests(unittest.TestCase):
    def request(self, fields=("title",)):
        return TranslationRequest("ko", "en", tuple(fields), model="fixture-model")

    def test_passthrough_retains_source_and_materializes_document_values(self):
        records = [{"record_id": "Q-1", "source_ref": "qms://issue/Q-1", "title": "오류"}]
        batch = translate_records(records, PassthroughTranslator(), self.request())
        self.assertEqual(batch.records[0]["title"], "오류")
        self.assertEqual(batch.records[0]["translation"]["fields"]["title"]["source"], "오류")
        self.assertEqual(batch.document_records[0]["title"], "오류")
        self.assertTrue(batch.manifest_fragment()["local_transport_only"])
        self.assertEqual(batch.manifest_fragment()["transport_scope"], "in-process")

    def test_translation_requires_a_local_adapter(self):
        with self.assertRaises(TranslationError):
            translate_records([{"title": "x"}], _RemoteTranslator(), self.request())

    def test_missing_field_and_empty_model_result_fail_closed(self):
        with self.assertRaises(TranslationError):
            translate_records([{"record_id": "Q-1"}], PassthroughTranslator(), self.request())

        class EmptyTranslator(PassthroughTranslator):
            def translate_text(self, text, request):
                return ""

        with self.assertRaises(TranslationError):
            translate_records([{"title": "x"}], EmptyTranslator(), self.request())

    def test_ollama_endpoint_is_loopback_only_and_parses_json_contract(self):
        with self.assertRaises(ValueError):
            OllamaTranslator("model", endpoint="https://example.com")
        with patch.dict(os.environ, {"OLLAMA_NO_CLOUD": "1"}):
            translator = OllamaTranslator("fixture-model", endpoint="http://127.0.0.1:11434")
        payload = {"message": {"content": json.dumps({"translation": "translated"})}}
        with patch.dict(os.environ, {"OLLAMA_NO_CLOUD": "1"}):
            with patch("translation_pipeline.urlopen", return_value=_FakeResponse(payload)) as mocked:
                result = translator.translate_text("source", self.request())
        self.assertEqual(result, "translated")
        request = mocked.call_args.args[0]
        body = json.loads(request.data.decode("utf-8"))
        self.assertEqual(request.full_url, "http://127.0.0.1:11434/api/chat")
        self.assertTrue(body["stream"] is False)
        self.assertEqual(body["model"], "fixture-model")
        self.assertEqual(translator.model_execution_location, "unverified-server")

    def test_ollama_invalid_response_is_rejected(self):
        with patch.dict(os.environ, {"OLLAMA_NO_CLOUD": "1"}):
            translator = OllamaTranslator("fixture-model")
        payload = {"message": {"content": "not-json"}}
        with patch.dict(os.environ, {"OLLAMA_NO_CLOUD": "1"}):
            with patch("translation_pipeline.urlopen", return_value=_FakeResponse(payload)):
                with self.assertRaises(TranslationError):
                    translator.translate_text("source", self.request())

    def test_local_opener_has_no_proxy_and_rejects_redirects(self):
        proxy_handlers = [handler for handler in translation_pipeline._LOCAL_OPENER.handlers if isinstance(handler, ProxyHandler)]
        self.assertEqual(proxy_handlers, [])
        with self.assertRaises(TranslationError):
            translation_pipeline._NoRedirectHandler().redirect_request(None, "http://127.0.0.1:11434", None)

    def test_ollama_requires_cloud_disabled_and_rejects_model_mismatch_or_extra_keys(self):
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError):
                OllamaTranslator("fixture-model")
        with patch.dict(os.environ, {"OLLAMA_NO_CLOUD": "1"}):
            with self.assertRaises(ValueError):
                OllamaTranslator("qwen3:cloud")
            translator = OllamaTranslator("fixture-model")
            with self.assertRaises(TranslationError):
                translator.translate_text("source", TranslationRequest("ko", "en", ("title",), model="other"))
            payload = {"message": {"content": json.dumps({"translation": "x", "extra": "reject"})}}
            with patch("translation_pipeline.urlopen", return_value=_FakeResponse(payload)):
                with self.assertRaises(TranslationError):
                    translator.translate_text("source", self.request())


if __name__ == "__main__":
    unittest.main()
