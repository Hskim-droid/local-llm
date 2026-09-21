"""Backend-neutral contracts for discovering and using local UI surfaces.

The control plane should plan against these records rather than against a
Playwright locator, an AXUIElement, or a Windows UI Automation object. Each
platform adapter can expose the same observation/action/evidence shape while
keeping its native dependency local to that adapter.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
import hashlib
import json
from typing import Any, Protocol
from uuid import uuid4


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


@dataclass(frozen=True)
class UiNode:
    """A normalized, actionable item in an adapter's UI graph."""

    node_id: str
    role: str
    name: str
    state: dict[str, Any] = field(default_factory=dict)
    parent_id: str | None = None
    actions: tuple[str, ...] = ()
    source_ref: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        result = asdict(self)
        result["actions"] = list(self.actions)
        return result


@dataclass(frozen=True)
class UiObservation:
    """A point-in-time view of a surface and its advertised capabilities.

    ``observation_hash`` identifies equivalent content. ``observation_id`` is a
    per-capture generation and must be attached to every action request, even
    when two consecutive captures have the same content hash.
    """

    backend: str
    surface: str
    title: str
    url: str
    nodes: tuple[UiNode, ...]
    capabilities: tuple[str, ...]
    captured_at: str = field(default_factory=utc_now)
    schema_version: int = 1
    observation_id: str = field(default_factory=lambda: f"obs-{uuid4().hex}")
    observation_hash: str = ""

    def __post_init__(self) -> None:
        stable = {
            "backend": self.backend,
            "surface": self.surface,
            "title": self.title,
            "url": self.url,
            "nodes": [node.to_dict() for node in self.nodes],
            "capabilities": list(self.capabilities),
        }
        digest = hashlib.sha256(json.dumps(stable, sort_keys=True, ensure_ascii=False).encode()).hexdigest()
        if not self.observation_hash:
            object.__setattr__(self, "observation_hash", digest[:16])

    def to_dict(self) -> dict[str, Any]:
        result = asdict(self)
        result["nodes"] = [node.to_dict() for node in self.nodes]
        result["capabilities"] = list(self.capabilities)
        return result


@dataclass(frozen=True)
class ActionRequest:
    """A planned action pinned to a required point-in-time observation."""

    action: str
    target_id: str
    reason: str
    observation_id: str
    expected_capabilities: tuple[str, ...] = ()


@dataclass(frozen=True)
class ActionReceipt:
    """Evidence that an adapter accepted and executed one action."""

    action_id: str
    action: str
    target_id: str
    backend: str
    reason: str
    executed_at: str = field(default_factory=utc_now)
    accepted: bool = True
    evidence: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


class SurfaceAdapter(Protocol):
    """Minimal interface consumed by a platform-neutral planner."""

    backend_name: str
    surface_kind: str

    def observe(self) -> UiObservation:
        ...

    def execute(self, request: ActionRequest, target: Any | None = None) -> ActionReceipt:
        ...

    def close(self) -> None:
        ...


def new_action_id() -> str:
    return f"act-{uuid4().hex}"
