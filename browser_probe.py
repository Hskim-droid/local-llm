#!/usr/bin/env python3
"""Role-based browser discovery and read-only table extraction.

The probe deliberately knows no menu path or CSS selector. It receives a task
contract with semantic aliases, inspects the accessible controls exposed by the
page, and stops when the target is ambiguous or unsupported.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from surface_adapter import ActionReceipt, ActionRequest, SurfaceAdapter, UiNode, UiObservation, new_action_id

try:
    from playwright.sync_api import Page, sync_playwright
except ImportError as exc:  # pragma: no cover - setup path
    Page = Any
    sync_playwright = None
    IMPORT_ERROR = exc
else:
    IMPORT_ERROR = None


class ProbeError(RuntimeError):
    """The surface cannot be safely resolved from the task contract."""


def require_dependencies() -> None:
    if IMPORT_ERROR is not None:
        raise RuntimeError(
            "browser probe dependencies are missing; install with "
            "python3 -m pip install -r requirements-fixture.txt "
            "and then run `python3 -m playwright install chromium`"
        ) from IMPORT_ERROR


Candidate = tuple[int, Any, str, str]


def normalize(value: str) -> str:
    return re.sub(r"[^\w가-힣]+", " ", value.casefold(), flags=re.UNICODE).strip()


def name_of(locator: Any) -> str:
    aria = locator.get_attribute("aria-label") or ""
    if aria.strip():
        return aria.strip()
    try:
        return locator.inner_text(timeout=500).strip()
    except Exception:
        return ""


def visible(locator: Any) -> bool:
    try:
        return locator.is_visible()
    except Exception:
        return False


def aliases_from(task: dict[str, Any], key: str) -> list[str]:
    values = task.get(key, [])
    if not isinstance(values, list) or not all(isinstance(value, str) for value in values):
        raise ProbeError(f"task.{key} must be a list of strings")
    return values


def score_name(name: str, aliases: list[str]) -> int:
    candidate = normalize(name)
    if not candidate:
        return 0
    scores = []
    for alias in aliases:
        normalized = normalize(alias)
        if not normalized:
            continue
        if candidate == normalized:
            scores.append(100 + len(normalized))
        elif normalized in candidate:
            scores.append(60 + len(normalized))
        else:
            alias_tokens = set(normalized.split())
            candidate_tokens = set(candidate.split())
            overlap = len(alias_tokens & candidate_tokens)
            if overlap:
                scores.append(10 * overlap + overlap / max(len(alias_tokens), 1))
    return max(scores, default=0)


def ui_node_id(role: str, index: int) -> str:
    return f"playwright-aria:{role}:{index}"


def semantic_candidates(page: Page, roles: tuple[str, ...], aliases: list[str]) -> list[Candidate]:
    candidates: list[Candidate] = []
    for role in roles:
        locator = page.get_by_role(role)
        for index in range(locator.count()):
            item = locator.nth(index)
            if not visible(item):
                continue
            name = name_of(item)
            score = score_name(name, aliases)
            if score:
                candidates.append((score, item, name, ui_node_id(role, index)))
    return sorted(candidates, key=lambda item: item[0], reverse=True)


def choose_unique(candidates: list[Candidate], purpose: str) -> tuple[Any, str, str]:
    if not candidates:
        raise ProbeError(f"no candidate found for {purpose}")
    best_score = candidates[0][0]
    best = [candidate for candidate in candidates if candidate[0] == best_score]
    if len(best) != 1:
        names = [candidate[2] for candidate in best]
        raise ProbeError(f"ambiguous {purpose}: {names}")
    return best[0][1], best[0][2], best[0][3]


class PlaywrightAriaAdapter:
    """SurfaceAdapter implementation for a browser accessibility tree."""

    backend_name = "playwright-aria"
    surface_kind = "browser"

    def __init__(self, page: Page):
        self.page = page
        self._node_refs: dict[str, tuple[Any, str, str]] = {}
        self._observation: UiObservation | None = None

    def observe(self) -> UiObservation:
        node_refs: dict[str, tuple[Any, str, str]] = {}
        nodes: list[UiNode] = []
        try:
            for role in ("button", "link", "menuitem", "heading", "textbox", "table"):
                locator = self.page.get_by_role(role)
                for index in range(locator.count()):
                    item = locator.nth(index)
                    if not visible(item):
                        continue
                    name = name_of(item)
                    state: dict[str, Any] = {}
                    actions: tuple[str, ...] = ()
                    if role in {"button", "link", "menuitem"}:
                        state["enabled"] = item.is_enabled()
                        if state["enabled"]:
                            actions = ("click",)
                    has_popup = item.get_attribute("aria-haspopup")
                    if has_popup:
                        state["has_popup"] = has_popup
                    expanded = item.get_attribute("aria-expanded")
                    if expanded is not None:
                        state["expanded"] = expanded == "true"
                    node_id = ui_node_id(role, index)
                    nodes.append(
                        UiNode(
                            node_id=node_id,
                            role=role,
                            name=name,
                            state=state,
                            actions=actions,
                            source_ref={"role": role, "index": index},
                        )
                    )
                    if actions:
                        node_refs[node_id] = (item, role, name)
        except Exception:
            self._node_refs = {}
            self._observation = None
            raise
        capabilities = ["observe_tree"]
        if any(node.role == "table" for node in nodes):
            capabilities.append("read_table")
        if any(node.role in {"button", "link", "menuitem"} for node in nodes):
            capabilities.append("navigate")
        observation = UiObservation(
            backend=self.backend_name,
            surface=self.surface_kind,
            title=self.page.title(),
            url=self.page.url,
            nodes=tuple(nodes),
            capabilities=tuple(capabilities),
        )
        self._node_refs = node_refs
        self._observation = observation
        return observation

    def execute(self, request: ActionRequest, target: Any | None = None) -> ActionReceipt:
        if request.action != "click":
            raise ProbeError(f"unsupported browser action: {request.action}")
        if self._observation is None:
            raise ProbeError("browser action requires a current observation")
        if request.observation_id != self._observation.observation_id:
            raise ProbeError("action observation is stale; observe the surface again")
        if request.expected_capabilities:
            if self._observation is None:
                raise ProbeError("browser action requires a current observation")
            available = set(self._observation.capabilities)
            missing = set(request.expected_capabilities) - available
            if missing:
                raise ProbeError(f"surface lacks capabilities: {sorted(missing)}")
        if target is not None:
            raise ProbeError("browser actions must resolve the target from its observed node id")
        if request.target_id not in self._node_refs:
            raise ProbeError("target is missing from the latest browser observation")
        target, expected_role, expected_name = self._node_refs[request.target_id]
        if not visible(target):
            raise ProbeError(f"target is not visible: {request.target_id}")
        if not target.is_enabled():
            raise ProbeError(f"target is disabled: {request.target_id}")
        if name_of(target) != expected_name:
            raise ProbeError(f"target changed since observation: {request.target_id}")
        if self.page.get_by_role(expected_role, name=expected_name, exact=True).count() != 1:
            raise ProbeError(f"target is no longer unique: {request.target_id}")
        target.click()
        return ActionReceipt(
            action_id=new_action_id(),
            action=request.action,
            target_id=request.target_id,
            backend=self.backend_name,
            reason=request.reason,
            evidence={
                "surface": self.surface_kind,
                "observation_id": self._observation.observation_id,
            },
        )

    def close(self) -> None:
        self._node_refs = {}
        self._observation = None
        return None


def observe(page: Page) -> dict[str, Any]:
    """Compatibility helper returning the normalized observation as a dict."""

    return PlaywrightAriaAdapter(page).observe().to_dict()


def open_navigation(
    page: Page,
    task: dict[str, Any],
    actions: list[dict[str, Any]],
    adapter: SurfaceAdapter,
    observation_id: str | None = None,
) -> None:
    candidates = semantic_candidates(page, ("button", "link"), aliases_from(task, "navigation_terms"))
    target, name, target_id = choose_unique(candidates, "navigation trigger")
    receipt = adapter.execute(
        ActionRequest(
            action="click",
            target_id=target_id,
            reason="open navigation",
            observation_id=observation_id,
        )
    )
    actions.append(receipt.to_dict())


def visible_table_with_fields(
    page: Page,
    fields: dict[str, list[str]],
    table_terms: list[str],
) -> tuple[Any, dict[str, int]] | None:
    tables = page.get_by_role("table")
    table_matches: list[tuple[int, Any, dict[str, int]]] = []
    for index in range(tables.count()):
        table = tables.nth(index)
        if not visible(table):
            continue
        table_name = table.get_attribute("aria-label") or ""
        table_score = score_name(table_name, table_terms) if table_terms else 1
        # A table label must contain a complete task term. A single shared
        # token such as "issues" is too weak and could select a completed or
        # otherwise unrelated table before the requested view is opened.
        if table_terms and table_score < 60:
            continue
        headers = [text.strip() for text in table.get_by_role("columnheader").all_text_contents()]
        mapping: dict[str, int] = {}
        used_positions: set[int] = set()
        for field, aliases in fields.items():
            column_matches = [(position, score_name(header, aliases)) for position, header in enumerate(headers)]
            column_matches = [(position, score) for position, score in column_matches if score]
            if not column_matches:
                break
            best_score = max(score for _, score in column_matches)
            best = [position for position, score in column_matches if score == best_score]
            if len(best) != 1:
                raise ProbeError(f"ambiguous column for {field}: {headers}")
            if best[0] in used_positions:
                raise ProbeError(f"multiple fields map to column {headers[best[0]]}: {field}")
            mapping[field] = best[0]
            used_positions.add(best[0])
        if len(mapping) == len(fields):
            table_matches.append((table_score, table, mapping))
    if table_matches:
        best_score = max(score for score, _, _ in table_matches)
        best = [match for match in table_matches if match[0] == best_score]
        if len(best) != 1:
            raise ProbeError(f"ambiguous target table: {len(best)} tables share the best task match")
        _, table, mapping = best[0]
        return table, mapping
    return None


def extract_table(table: Any, mapping: dict[str, int], fields: dict[str, list[str]], identity_field: str) -> list[dict[str, Any]]:
    rows = table.get_by_role("row")
    records: list[dict[str, Any]] = []
    seen: set[str] = set()
    for index in range(1, rows.count()):
        values = [value.strip() for value in rows.nth(index).get_by_role("cell").all_text_contents()]
        record: dict[str, Any] = {}
        for field in fields:
            position = mapping[field]
            if position >= len(values) or not values[position]:
                raise ProbeError(f"blank or missing {field} in row {index}")
            record[field] = values[position]
        record_id = record[identity_field]
        if record_id in seen:
            raise ProbeError(f"duplicate record id: {record_id}")
        seen.add(record_id)
        record["source_ref"] = {"backend": "playwright-aria", "table_row": index}
        records.append(record)
    if not records:
        raise ProbeError("target table has no records")
    return records


def run_task(html_path: Path, task: dict[str, Any]) -> dict[str, Any]:
    require_dependencies()
    if not isinstance(task.get("fields"), dict) or not task["fields"]:
        raise ProbeError("task.fields must be a non-empty object")
    fields = task["fields"]
    identity_field = task.get("identity_field")
    if not isinstance(identity_field, str) or identity_field not in fields:
        raise ProbeError("task.identity_field must name one of task.fields")
    if not all(isinstance(key, str) and isinstance(value, list) for key, value in fields.items()):
        raise ProbeError("task.fields values must be alias lists")
    table_terms = aliases_from(task, "table_terms")
    forbidden_terms = aliases_from(task, "forbidden_terms")
    if not table_terms:
        raise ProbeError("task.table_terms must identify the requested table")
    if not forbidden_terms:
        raise ProbeError("task.forbidden_terms must declare blocked actions")
    actions: list[dict[str, Any]] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        try:
            page = browser.new_page()
            page.goto(html_path.resolve().as_uri(), wait_until="load")
            adapter = PlaywrightAriaAdapter(page)
            before_observation = adapter.observe()
            before = before_observation.to_dict()
            open_navigation(page, task, actions, adapter, before_observation.observation_id)
            target_terms = aliases_from(task, "target_terms")
            table_result = visible_table_with_fields(page, fields, table_terms)
            for _ in range(5):
                if table_result is not None:
                    break
                observation = adapter.observe()
                # The navigation trigger itself can contain words such as
                # "open". Only menu items and links may satisfy the task
                # target; the trigger is handled once above.
                candidates = semantic_candidates(page, ("menuitem", "link"), target_terms)
                # A parent category such as "Inspection" can match the
                # semantic target while still being only an expandable
                # container. Prefer a leaf action when one is visible.
                leaf_candidates = [
                    candidate
                    for candidate in candidates
                    if not candidate[1].get_attribute("aria-haspopup")
                ]
                if leaf_candidates:
                    candidates = leaf_candidates
                unsafe_candidates = [
                    candidate for candidate in candidates if score_name(candidate[2], forbidden_terms)
                ]
                if unsafe_candidates and len(unsafe_candidates) == len(candidates):
                    raise ProbeError(f"target navigation is forbidden: {[candidate[2] for candidate in unsafe_candidates]}")
                candidates = [candidate for candidate in candidates if candidate not in unsafe_candidates]
                if candidates:
                    target, name, target_id = choose_unique(candidates, "target navigation item")
                else:
                    expandable: list[Candidate] = []
                    for role in ("menuitem", "button"):
                        locator = page.get_by_role(role)
                        for index in range(locator.count()):
                            item = locator.nth(index)
                            if (
                                visible(item)
                                and item.get_attribute("aria-haspopup")
                                and item.get_attribute("aria-expanded") != "true"
                            ):
                                name = name_of(item)
                                if not score_name(name, forbidden_terms):
                                    expandable.append((1, item, name, ui_node_id(role, index)))
                    target, name, target_id = choose_unique(expandable, "expandable navigation item")
                receipt = adapter.execute(
                    ActionRequest(
                        action="click",
                        target_id=target_id,
                        reason="semantic navigation",
                        observation_id=observation.observation_id,
                    )
                )
                actions.append(receipt.to_dict())
                page.wait_for_timeout(25)
                table_result = visible_table_with_fields(page, fields, table_terms)
            if table_result is None:
                raise ProbeError("no visible table matched task fields after navigation")
            table, mapping = table_result
            records = extract_table(table, mapping, fields, identity_field)
            after = adapter.observe().to_dict()
            return {
                "task_id": task.get("task_id", "unnamed"),
                "goal": task.get("goal", ""),
                "observation_before": before,
                "actions": actions,
                "observation_after": after,
                "field_mapping": mapping,
                "records": records,
                "checks": [
                    {"name": "unique_target_navigation", "passed": True},
                    {"name": "required_fields_present", "passed": True},
                    {"name": "unique_record_ids", "passed": True},
                ],
            }
        finally:
            browser.close()


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Discover and read a browser surface using a task contract")
    parser.add_argument("--html", required=True)
    parser.add_argument("--task", required=True, help="JSON task contract")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        task = json.loads(Path(args.task).read_text(encoding="utf-8"))
        result = run_task(Path(args.html), task)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0
    except (OSError, RuntimeError, ValueError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
