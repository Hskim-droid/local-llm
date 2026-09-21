#!/usr/bin/env python3
"""Translate extracted records with an explicitly local model boundary.

The UI adapters only collect evidence. This module is the next stage: it keeps
the extracted values, asks a local translator for selected fields, and exposes
both an audit-friendly translation fragment and flat records for a renderer.
The Ollama transport is deliberately loopback-only; there is no cloud fallback.
"""

from __future__ import annotations

import argparse
from dataclasses import dataclass
import json
import os
import re
from pathlib import Path
from typing import Any, Mapping, Protocol, Sequence
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener


class TranslationError(RuntimeError):
    """The local translation stage cannot produce a verified result."""


class _NoRedirectHandler(HTTPRedirectHandler):
    def redirect_request(self, *_args, **_kwargs):
        raise TranslationError("local model endpoint returned a redirect")


# Do not inherit proxy variables from the host and do not follow a redirect to
# an endpoint that was not checked by _loopback_endpoint.
_LOCAL_OPENER = build_opener(ProxyHandler({}), _NoRedirectHandler())


def urlopen(request_obj: Request, timeout: float):
    """Open a checked loopback request without proxy or redirect handling."""
    return _LOCAL_OPENER.open(request_obj, timeout=timeout)


@dataclass(frozen=True)
class TranslationRequest:
    source_language: str
    target_language: str
    fields: tuple[str, ...]
    model: str | None = None
    glossary: Mapping[str, str] | None = None

    def __post_init__(self) -> None:
        for name, language in (
            ("source_language", self.source_language),
            ("target_language", self.target_language),
        ):
            if not isinstance(language, str) or not language.strip():
                raise ValueError(f"{name} must be a non-empty language tag")
        if not self.fields or len(set(self.fields)) != len(self.fields):
            raise ValueError("fields must contain at least one unique field name")
        if any(not isinstance(field, str) or not field.strip() for field in self.fields):
            raise ValueError("fields must contain non-empty names")
        if self.model is not None and not self.model.strip():
            raise ValueError("model must be non-empty when provided")


class TextTranslator(Protocol):
    """Small adapter contract for a local translation engine."""

    backend_name: str
    model_name: str | None
    local_only: bool
    transport_scope: str
    model_execution_location: str

    def translate_text(self, text: str, request: TranslationRequest) -> str:
        ...


class PassthroughTranslator:
    """Deterministic fixture translator used when no model is available."""

    backend_name = "passthrough-fixture"
    model_name = None
    local_only = True
    transport_scope = "in-process"
    model_execution_location = "in-process"

    def translate_text(self, text: str, request: TranslationRequest) -> str:
        if not isinstance(text, str):
            raise TranslationError("passthrough translator received a non-string value")
        return text


def _loopback_endpoint(endpoint: str) -> str:
    if not isinstance(endpoint, str) or not endpoint.strip():
        raise ValueError("local model endpoint must be a non-empty URL")
    parsed = urlsplit(endpoint)
    if parsed.scheme not in {"http", "https"}:
        raise ValueError("local model endpoint must use http or https")
    if parsed.username or parsed.password:
        raise ValueError("credentials are not allowed in a local model endpoint")
    hostname = (parsed.hostname or "").lower()
    if hostname not in {"localhost", "127.0.0.1", "::1"}:
        raise ValueError("local model endpoint must resolve to localhost")
    if parsed.query or parsed.fragment:
        raise ValueError("local model endpoint must not contain a query or fragment")
    return endpoint.rstrip("/")


class OllamaTranslator:
    """Call an Ollama-compatible chat endpoint on the local machine only."""

    backend_name = "ollama-local"
    local_only = True
    transport_scope = "loopback-only"
    model_execution_location = "unverified-server"

    def __init__(self, model: str, endpoint: str = "http://127.0.0.1:11434", timeout: float = 60.0) -> None:
        if not isinstance(model, str) or not model.strip():
            raise ValueError("a local model name is required")
        if re.search(r"(^|[-_:/])cloud($|[-_:/])", model.strip().casefold()):
            raise ValueError("cloud model names are not allowed by the local translation adapter")
        if timeout <= 0:
            raise ValueError("timeout must be positive")
        if os.environ.get("OLLAMA_NO_CLOUD") != "1":
            raise ValueError("set OLLAMA_NO_CLOUD=1 before using the Ollama local backend")
        self.model_name = model.strip()
        self.endpoint = _loopback_endpoint(endpoint)
        self.timeout = timeout

    @property
    def chat_url(self) -> str:
        if self.endpoint.endswith("/api"):
            return f"{self.endpoint}/chat"
        return f"{self.endpoint}/api/chat"

    def translate_text(self, text: str, request: TranslationRequest) -> str:
        if not isinstance(text, str):
            raise TranslationError("local translator received a non-string value")
        if request.model and request.model != self.model_name:
            raise TranslationError("translation request model does not match the configured local model")
        if not text.strip():
            return ""
        glossary = ""
        if request.glossary:
            glossary = "\nGlossary (keep these terms consistent): " + json.dumps(
                dict(request.glossary), ensure_ascii=False, sort_keys=True
            )
        prompt = (
            f"Translate the source text from {request.source_language} to "
            f"{request.target_language}. Return one JSON object with exactly one "
            f"string property named translation. Preserve IDs, dates, numbers, "
            f"and line breaks. Do not add commentary.{glossary}\n\n"
            f"SOURCE TEXT:\n{text}"
        )
        body = {
            "model": request.model or self.model_name,
            "messages": [
                {
                    "role": "system",
                    "content": "You are a local document translation component. Follow the JSON contract exactly.",
                },
                {"role": "user", "content": prompt},
            ],
            "stream": False,
            "format": "json",
        }
        request_obj = Request(
            self.chat_url,
            data=json.dumps(body, ensure_ascii=False).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urlopen(request_obj, timeout=self.timeout) as response:
                raw = response.read(4 * 1024 * 1024 + 1)
        except (HTTPError, URLError, TimeoutError, OSError) as exc:
            raise TranslationError(f"local model request failed: {exc}") from exc
        if len(raw) > 4 * 1024 * 1024:
            raise TranslationError("local model response exceeded the size limit")
        try:
            envelope = json.loads(raw.decode("utf-8"))
            content = envelope["message"]["content"]
            result = json.loads(content)
            if not isinstance(result, dict) or set(result) != {"translation"}:
                raise TranslationError("local model response must contain exactly the translation field")
            translated = result["translation"]
        except (UnicodeDecodeError, json.JSONDecodeError, KeyError, TypeError) as exc:
            raise TranslationError("local model returned an invalid translation JSON response") from exc
        if not isinstance(translated, str):
            raise TranslationError("local model translation must be a string")
        return translated


@dataclass(frozen=True)
class TranslationBatch:
    source_language: str
    target_language: str
    backend: str
    model: str | None
    transport_scope: str
    model_execution_location: str
    fields: tuple[str, ...]
    records: list[dict[str, Any]]
    document_records: list[dict[str, Any]]
    items: list[dict[str, Any]]

    def manifest_fragment(self) -> dict[str, Any]:
        return {
            "source_language": self.source_language,
            "target_language": self.target_language,
            "backend": self.backend,
            "model": self.model,
            "local_transport_only": True,
            "transport_scope": self.transport_scope,
            "model_execution_location": self.model_execution_location,
            "fields": list(self.fields),
            "items": self.items,
        }


def translate_records(
    records: Sequence[Mapping[str, Any]],
    translator: TextTranslator,
    request: TranslationRequest,
) -> TranslationBatch:
    """Translate selected fields while retaining source values and evidence."""
    if not getattr(translator, "local_only", False):
        raise TranslationError("translation adapter must declare local_only=True")
    translated_records: list[dict[str, Any]] = []
    document_records: list[dict[str, Any]] = []
    items: list[dict[str, Any]] = []
    for index, record in enumerate(records):
        if not isinstance(record, Mapping):
            raise TranslationError(f"record {index + 1} is not an object")
        enriched = dict(record)
        document_record = dict(record)
        field_evidence: dict[str, dict[str, str]] = {}
        for field in request.fields:
            if field not in record:
                raise TranslationError(f"record {index + 1} is missing translation field: {field}")
            source = record[field]
            if not isinstance(source, str):
                raise TranslationError(f"record {index + 1} field {field} is not text")
            translated = translator.translate_text(source, request)
            if not isinstance(translated, str):
                raise TranslationError(f"translator returned a non-string value for record {index + 1} field {field}")
            if source.strip() and not translated.strip():
                raise TranslationError(f"translator returned empty text for record {index + 1} field {field}")
            field_evidence[field] = {"source": source, "translated": translated}
            document_record[field] = translated
        translation_fragment = {
            "source_language": request.source_language,
            "target_language": request.target_language,
            "backend": translator.backend_name,
            "model": translator.model_name,
            "fields": field_evidence,
        }
        enriched["translation"] = translation_fragment
        translated_records.append(enriched)
        document_records.append(document_record)
        items.append(
            {
                "record_id": record.get("record_id"),
                "source_ref": record.get("source_ref"),
                "fields": field_evidence,
            }
        )
    return TranslationBatch(
        source_language=request.source_language,
        target_language=request.target_language,
        backend=translator.backend_name,
        model=translator.model_name,
        transport_scope=getattr(translator, "transport_scope", "unspecified"),
        model_execution_location=getattr(translator, "model_execution_location", "unverified"),
        fields=request.fields,
        records=translated_records,
        document_records=document_records,
        items=items,
    )


def build_translator(backend: str, *, model: str | None = None, endpoint: str | None = None) -> TextTranslator:
    if backend == "passthrough":
        return PassthroughTranslator()
    if backend == "ollama":
        if not model:
            raise ValueError("--model is required for the ollama backend")
        return OllamaTranslator(model=model, endpoint=endpoint or "http://127.0.0.1:11434")
    raise ValueError(f"unknown translation backend: {backend}")


def _load_records(path: Path) -> list[dict[str, Any]]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if isinstance(payload, dict):
        payload = payload.get("records")
    if not isinstance(payload, list):
        raise ValueError("input JSON must be a records array or an object with records")
    if not all(isinstance(record, dict) for record in payload):
        raise ValueError("input records must be JSON objects")
    return payload


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Translate extracted records through a local model boundary")
    parser.add_argument("--input", required=True, type=Path, help="JSON records array or object with a records array")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--from", dest="source_language", required=True)
    parser.add_argument("--to", dest="target_language", required=True)
    parser.add_argument("--fields", required=True, help="comma-separated record fields to translate")
    parser.add_argument("--backend", choices=("passthrough", "ollama"), default="ollama")
    parser.add_argument("--model")
    parser.add_argument("--endpoint", default="http://127.0.0.1:11434")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        fields = tuple(field.strip() for field in args.fields.split(",") if field.strip())
        request = TranslationRequest(args.source_language, args.target_language, fields, model=args.model)
        translator = build_translator(args.backend, model=args.model, endpoint=args.endpoint)
        batch = translate_records(_load_records(args.input), translator, request)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(
            json.dumps(
                {"translation": batch.manifest_fragment(), "records": batch.records, "document_records": batch.document_records},
                ensure_ascii=False,
                indent=2,
            ),
            encoding="utf-8",
        )
        print(json.dumps({"output": str(args.output), "records": len(batch.records)}, ensure_ascii=False))
        return 0
    except (OSError, ValueError, TranslationError, json.JSONDecodeError) as exc:
        print(f"error: {exc}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
