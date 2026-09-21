"""Stable request types for the local-first document workflow SDK."""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from typing import Any, Mapping


INPUT_KINDS = frozenset({
    "file",
    "audio",
    "video",
    "browser",
    "native_app",
    "local_system",
    "web_app",
})
OUTPUT_FORMATS = frozenset({"docx", "xlsx", "pptx"})


@dataclass(frozen=True)
class InputRef:
    """A bounded local input or an adapter-owned surface reference.

    ``locator`` is deliberately opaque to the core. File adapters may use a
    local path; browser and native adapters may use a private handle. The
    public contract never stores cookies, credentials, or raw source bytes.
    """

    kind: str
    locator: str
    media_type: str | None = None
    source_language: str | None = None
    scope: Mapping[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if self.kind not in INPUT_KINDS:
            raise ValueError(f"unsupported input kind: {self.kind}")
        if not isinstance(self.locator, str) or not self.locator.strip():
            raise ValueError("input locator must be a non-empty string")
        if not isinstance(self.scope, Mapping):
            raise ValueError("input scope must be an object")
        if self.kind in {"browser", "web_app"} and not self.scope.get("allowed_domains"):
            raise ValueError("browser and web_app inputs require scope.allowed_domains")

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> "InputRef":
        if not isinstance(value, Mapping):
            raise ValueError("input must be an object")
        return cls(
            kind=str(value.get("kind", "")),
            locator=str(value.get("locator", "")),
            media_type=value.get("media_type"),
            source_language=value.get("source_language"),
            scope=value.get("scope", {}),
        )

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class WorkflowPolicy:
    """Safety policy attached to every SDK request."""

    mode: str = "draft_only"
    approval: str = "required"
    allow_remote_write: bool = False
    allow_external_send: bool = False

    def __post_init__(self) -> None:
        if self.mode != "draft_only":
            raise ValueError("the public SDK only permits draft_only mode")
        if self.approval != "required":
            raise ValueError("human approval is required for workflow output")
        if self.allow_remote_write or self.allow_external_send:
            raise ValueError("remote writes and external sends are outside the public SDK")

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> "WorkflowPolicy":
        if not isinstance(value, Mapping):
            raise ValueError("policy must be an object")
        return cls(
            mode=str(value.get("mode", "draft_only")),
            approval=str(value.get("approval", "required")),
            allow_remote_write=bool(value.get("allow_remote_write", False)),
            allow_external_send=bool(value.get("allow_external_send", False)),
        )

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class WorkflowRequest:
    """Structured input passed by a coding tool or a local scheduler."""

    task: str
    inputs: tuple[InputRef, ...]
    output_format: str
    source_language: str = "auto"
    target_language: str | None = None
    policy: WorkflowPolicy = field(default_factory=WorkflowPolicy)

    def __post_init__(self) -> None:
        if not isinstance(self.task, str) or not self.task.strip():
            raise ValueError("task must be a non-empty string")
        inputs = tuple(self.inputs)
        if not inputs:
            raise ValueError("at least one input is required")
        if any(not isinstance(item, InputRef) for item in inputs):
            raise ValueError("inputs must contain InputRef objects")
        object.__setattr__(self, "inputs", inputs)
        if self.output_format not in OUTPUT_FORMATS:
            raise ValueError(f"unsupported output format: {self.output_format}")
        if not isinstance(self.source_language, str) or not self.source_language.strip():
            raise ValueError("source_language must be a non-empty language tag")
        if self.target_language is not None and not self.target_language.strip():
            raise ValueError("target_language must be non-empty when provided")
        if not isinstance(self.policy, WorkflowPolicy):
            raise ValueError("policy must be a WorkflowPolicy")

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> "WorkflowRequest":
        if not isinstance(value, Mapping):
            raise ValueError("workflow request must be an object")
        output = value.get("output", {})
        if not isinstance(output, Mapping):
            raise ValueError("output must be an object")
        raw_inputs = value.get("inputs", [])
        if not isinstance(raw_inputs, (list, tuple)):
            raise ValueError("inputs must be an array")
        return cls(
            task=str(value.get("task", "")),
            inputs=tuple(InputRef.from_mapping(item) for item in raw_inputs),
            output_format=str(output.get("format", "")),
            source_language=str(value.get("source_language", "auto")),
            target_language=value.get("target_language"),
            policy=WorkflowPolicy.from_mapping(value.get("policy", {})),
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "task": self.task,
            "inputs": [item.to_dict() for item in self.inputs],
            "output": {"format": self.output_format},
            "source_language": self.source_language,
            "target_language": self.target_language,
            "policy": self.policy.to_dict(),
        }
