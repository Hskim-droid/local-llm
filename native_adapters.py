"""Optional native accessibility adapters.

The public fixture does not install these platform SDK bindings. Each adapter
loads its dependency only when constructed, so a browser-only installation
continues to work on every platform. Native adapters intentionally require an
explicit root element/window; they never crawl the whole desktop by default.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass
import importlib.util
import os
import sys
from typing import Any

from surface_adapter import ActionReceipt, ActionRequest, UiNode, UiObservation, new_action_id


class NativeAdapterUnavailable(RuntimeError):
    """The optional platform binding or an explicit UI root is unavailable."""


AX_PERMISSION_ERROR = -25211
AX_OPTIONAL_ERRORS = {-25205, -25212, -25213}


@dataclass(frozen=True)
class BackendStatus:
    name: str
    surface: str
    available: bool
    reason: str

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


def _module_available(name: str) -> bool:
    return importlib.util.find_spec(name) is not None


def backend_status() -> list[BackendStatus]:
    """Report optional backend availability without importing native bindings."""

    statuses = [
        BackendStatus(
            name="playwright-aria",
            surface="browser",
            available=_module_available("playwright"),
            reason="installed" if _module_available("playwright") else "install playwright",
        )
    ]
    if sys.platform == "darwin":
        statuses.append(
            BackendStatus(
                name="macos-ax",
                surface="native",
                available=_module_available("Quartz") or _module_available("ApplicationServices"),
                reason="installed" if (_module_available("Quartz") or _module_available("ApplicationServices")) else "install pyobjc-framework-Quartz and grant Accessibility permission",
            )
        )
    else:
        statuses.append(BackendStatus("macos-ax", "native", False, "requires macOS"))
    if os.name == "nt":
        statuses.append(
            BackendStatus(
                name="windows-uia",
                surface="native",
                available=_module_available("pywinauto"),
                reason="installed" if _module_available("pywinauto") else "install pywinauto",
            )
        )
    else:
        statuses.append(BackendStatus("windows-uia", "native", False, "requires Windows"))
    if sys.platform.startswith("linux"):
        statuses.append(BackendStatus("linux-atspi", "native", False, "adapter not bundled yet"))
    else:
        statuses.append(BackendStatus("linux-atspi", "native", False, "requires Linux"))
    return statuses


def _load_macos_api() -> Any:
    for module_name in ("Quartz", "ApplicationServices"):
        try:
            module = __import__(module_name)
        except ImportError:
            continue
        required = ("AXUIElementCopyAttributeValue", "AXUIElementPerformAction")
        if all(hasattr(module, name) for name in required):
            return module
    raise NativeAdapterUnavailable(
        "macOS AX binding is unavailable; install pyobjc-framework-Quartz and grant Accessibility permission"
    )


def _ax_value(api: Any, element: Any, attribute: str, *, optional: bool = True) -> Any:
    try:
        result = api.AXUIElementCopyAttributeValue(element, attribute, None)
    except Exception as exc:
        raise NativeAdapterUnavailable(f"macOS AX attribute read failed: {attribute}") from exc
    if isinstance(result, tuple) and len(result) == 2:
        error, value = result
        if error == AX_PERMISSION_ERROR:
            raise NativeAdapterUnavailable("macOS Accessibility permission denied (AX error -25211)")
        if error not in (None, 0) and not (optional and error in AX_OPTIONAL_ERRORS):
            raise NativeAdapterUnavailable(f"macOS AX attribute read failed: {attribute} (status {error})")
        if error not in (None, 0):
            return None
        return value
    return result


def _ax_children(api: Any, element: Any) -> list[Any]:
    children = _ax_value(api, element, "AXChildren", optional=True)
    if children is None:
        return []
    return list(children) if isinstance(children, (list, tuple)) else []


def _ax_action_name(action: str) -> str | None:
    return {
        "click": "AXPress",
        "press": "AXPress",
        "open_menu": "AXShowMenu",
        "expand": "AXShowMenu",
    }.get(action)


def _ax_action_names(api: Any, element: Any) -> list[str]:
    if not hasattr(api, "AXUIElementCopyActionNames"):
        return []
    try:
        result = api.AXUIElementCopyActionNames(element, None)
    except Exception as exc:
        raise NativeAdapterUnavailable("macOS AX action discovery failed") from exc
    if isinstance(result, tuple) and len(result) == 2:
        error, names = result
        if error == AX_PERMISSION_ERROR:
            raise NativeAdapterUnavailable("macOS Accessibility permission denied (AX error -25211)")
        if error not in (None, 0) and error not in AX_OPTIONAL_ERRORS:
            raise NativeAdapterUnavailable(f"macOS AX action discovery failed (status {error})")
        if error not in (None, 0):
            return []
        result = names
    return list(result) if isinstance(result, (list, tuple)) else []


def _check_expected_capabilities(adapter: Any, request: ActionRequest) -> None:
    observation = getattr(adapter, "_observation", None)
    if observation is None:
        raise NativeAdapterUnavailable("action requires a current observation")
    if request.observation_id != observation.observation_id:
        raise NativeAdapterUnavailable("action observation is stale; observe the surface again")
    if not request.expected_capabilities:
        return
    available = set(observation.capabilities)
    missing = set(request.expected_capabilities) - available
    if missing:
        raise NativeAdapterUnavailable(f"surface lacks capabilities: {sorted(missing)}")


class MacAXAdapter:
    """Read and act on an explicitly supplied macOS AXUIElement tree."""

    backend_name = "macos-ax"
    surface_kind = "native"

    def __init__(
        self,
        root: Any | None = None,
        *,
        pid: int | None = None,
        api: Any | None = None,
        title: str = "",
        max_depth: int = 8,
        max_nodes: int = 5000,
    ):
        self.api = api if api is not None else _load_macos_api()
        if root is None:
            if pid is not None and hasattr(self.api, "AXUIElementCreateApplication"):
                root = self.api.AXUIElementCreateApplication(pid)
            else:
                raise NativeAdapterUnavailable("macOS AX adapter needs an explicit root element or application PID")
        self.root = root
        self.title = title
        self.max_depth = max_depth
        self.max_nodes = max_nodes
        self._elements: dict[str, Any] = {}
        self._node_specs: dict[str, UiNode] = {}
        self._partial = False
        self._closed = False
        self._observation: UiObservation | None = None

    def observe(self) -> UiObservation:
        if self._closed:
            raise NativeAdapterUnavailable("macOS AX adapter is closed")
        self._elements = {}
        self._node_specs = {}
        self._partial = False
        self._observation = None
        elements: dict[str, Any] = {}
        node_specs: dict[str, UiNode] = {}
        partial = False
        nodes: list[UiNode] = []

        def visit(element: Any, path: str, parent_id: str | None, depth: int) -> None:
            nonlocal partial
            if len(nodes) >= self.max_nodes or depth > self.max_depth:
                partial = True
                return
            role = str(_ax_value(self.api, element, "AXRole", optional=False) or "unknown")
            name = str(
                _ax_value(self.api, element, "AXTitle")
                or _ax_value(self.api, element, "AXDescription")
                or ""
            ).strip()
            node_id = f"macos-ax:{path}"
            native_actions = _ax_action_names(self.api, element)
            normalized_actions = tuple(
                action
                for action in ("click", "open_menu")
                if _ax_action_name(action) in native_actions
            )
            state: dict[str, Any] = {}
            enabled = _ax_value(self.api, element, "AXEnabled")
            if enabled is not None:
                state["enabled"] = bool(enabled)
            value = _ax_value(self.api, element, "AXValue")
            if value not in (None, "") and not isinstance(value, (list, tuple, dict)):
                state["value"] = str(value)
            node = UiNode(
                node_id=node_id,
                role=role,
                name=name,
                state=state,
                parent_id=parent_id,
                actions=normalized_actions,
                source_ref={"path": path, "native_role": role},
            )
            nodes.append(node)
            elements[node_id] = element
            node_specs[node_id] = node
            for index, child in enumerate(_ax_children(self.api, element)):
                visit(child, f"{path}.{index}", node_id, depth + 1)

        try:
            visit(self.root, "0", None, 0)
        except Exception:
            self._elements = {}
            self._node_specs = {}
            self._partial = True
            self._observation = None
            raise
        self._elements = elements
        self._node_specs = node_specs
        self._partial = partial
        capabilities = ["observe_tree", "native_accessibility"]
        if self._partial:
            capabilities.append("partial_tree")
        if any(node.actions for node in nodes):
            capabilities.append("navigate")
        if any("value" in node.state for node in nodes):
            capabilities.append("read_value")
        observation = UiObservation(
            backend=self.backend_name,
            surface=self.surface_kind,
            title=self.title,
            url="",
            nodes=tuple(nodes),
            capabilities=tuple(capabilities),
        )
        self._observation = observation
        return observation

    def execute(self, request: ActionRequest, target: Any | None = None) -> ActionReceipt:
        if self._closed:
            raise NativeAdapterUnavailable("macOS AX adapter is closed")
        native_action = _ax_action_name(request.action)
        if native_action is None:
            raise NativeAdapterUnavailable(f"unsupported macOS AX action: {request.action}")
        _check_expected_capabilities(self, request)
        if target is not None:
            raise NativeAdapterUnavailable("native actions must use an observed node id")
        if self._partial:
            raise NativeAdapterUnavailable("cannot act on a partial AX tree")
        element = self._elements.get(request.target_id)
        node = self._node_specs.get(request.target_id)
        if element is None:
            raise NativeAdapterUnavailable("target is missing; observe the AX tree before acting")
        if node is None or request.action not in node.actions:
            raise NativeAdapterUnavailable("requested action is not advertised by the observed AX node")
        if node.state.get("enabled") is False:
            raise NativeAdapterUnavailable("target AX node is disabled")
        result = self.api.AXUIElementPerformAction(element, native_action)
        if isinstance(result, tuple):
            result = result[0]
        if result not in (None, 0):
            raise NativeAdapterUnavailable(f"macOS AX action failed with status {result}")
        return ActionReceipt(
            action_id=new_action_id(),
            action=request.action,
            target_id=request.target_id,
            backend=self.backend_name,
            reason=request.reason,
            evidence={
                "native_action": native_action,
                "observation_id": self._observation.observation_id,
            },
        )

    def close(self) -> None:
        self._elements.clear()
        self._node_specs.clear()
        self._partial = True
        self._closed = True
        self._observation = None


def _uia_name(control: Any) -> str:
    try:
        return str(control.window_text() or "").strip()
    except Exception:
        return ""


class WindowsUIAAdapter:
    """Read and act on an explicitly supplied pywinauto UIA root."""

    backend_name = "windows-uia"
    surface_kind = "native"

    def __init__(self, root: Any | None = None, *, api: Any | None = None, title: str = ""):
        if root is None:
            raise NativeAdapterUnavailable(
                "Windows UIA adapter requires an explicit window root; use pywinauto Desktop(...).window(...)"
            )
        if api is None:
            if not _module_available("pywinauto"):
                raise NativeAdapterUnavailable("Windows UIA binding is unavailable; install pywinauto")
            from pywinauto import Desktop

            api = Desktop
        self.api = api
        self.root = root
        self.title = title or _uia_name(root)
        self._controls: dict[str, Any] = {}
        self._node_specs: dict[str, UiNode] = {}
        self._closed = False
        self._observation: UiObservation | None = None

    def observe(self) -> UiObservation:
        if self._closed:
            raise NativeAdapterUnavailable("Windows UIA adapter is closed")
        self._controls = {}
        self._node_specs = {}
        self._observation = None
        controls = [self.root]
        try:
            controls.extend(self.root.descendants())
        except Exception as exc:
            self._controls = {}
            self._node_specs = {}
            raise NativeAdapterUnavailable("could not enumerate Windows UIA descendants") from exc
        nodes: list[UiNode] = []
        controls_cache: dict[str, Any] = {}
        node_specs: dict[str, UiNode] = {}
        try:
            for index, control in enumerate(controls):
                try:
                    info = control.element_info
                    native_role = str(getattr(info, "control_type", "unknown") or "unknown")
                    runtime_id = getattr(info, "runtime_id", None)
                    if isinstance(runtime_id, (list, tuple)):
                        stable = "-".join(str(part) for part in runtime_id)
                    else:
                        stable = str(runtime_id) if runtime_id else str(index)
                    node_id = f"windows-uia:{stable}"
                    enabled = bool(control.is_enabled())
                except Exception as exc:
                    raise NativeAdapterUnavailable(f"could not read Windows UIA control {index}") from exc
                state = {"enabled": enabled}
                can_invoke = callable(getattr(control, "invoke", None)) or (
                    native_role == "TabItem" and callable(getattr(control, "select", None))
                )
                actions = ("click",) if enabled and can_invoke else ()
                node = UiNode(
                    node_id=node_id,
                    role=native_role,
                    name=_uia_name(control),
                    state=state,
                    actions=actions,
                    source_ref={"index": index, "runtime_id": runtime_id},
                )
                nodes.append(node)
                controls_cache[node_id] = control
                node_specs[node_id] = node
        except Exception:
            self._controls = {}
            self._node_specs = {}
            self._observation = None
            raise
        self._controls = controls_cache
        self._node_specs = node_specs
        capabilities = ["observe_tree", "native_accessibility"]
        if any(node.actions for node in nodes):
            capabilities.append("navigate")
        observation = UiObservation(
            backend=self.backend_name,
            surface=self.surface_kind,
            title=self.title,
            url="",
            nodes=tuple(nodes),
            capabilities=tuple(capabilities),
        )
        self._observation = observation
        return observation

    def execute(self, request: ActionRequest, target: Any | None = None) -> ActionReceipt:
        if self._closed:
            raise NativeAdapterUnavailable("Windows UIA adapter is closed")
        if request.action != "click":
            raise NativeAdapterUnavailable(f"unsupported Windows UIA action: {request.action}")
        _check_expected_capabilities(self, request)
        if target is not None:
            raise NativeAdapterUnavailable("native actions must use an observed node id")
        control = self._controls.get(request.target_id)
        node = self._node_specs.get(request.target_id)
        if control is None:
            raise NativeAdapterUnavailable("target is missing; observe the UIA tree before acting")
        if node is None or request.action not in node.actions:
            raise NativeAdapterUnavailable("requested action is not advertised by the observed UIA node")
        if node.state.get("enabled") is False:
            raise NativeAdapterUnavailable("target UIA node is disabled")
        try:
            if node.role == "TabItem" and callable(getattr(control, "select", None)):
                control.select()
            elif callable(getattr(control, "invoke", None)):
                control.invoke()
            else:
                raise NativeAdapterUnavailable("UIA control has no Invoke or Select pattern")
        except Exception as exc:
            raise NativeAdapterUnavailable(f"Windows UIA click failed for {request.target_id}") from exc
        return ActionReceipt(
            action_id=new_action_id(),
            action=request.action,
            target_id=request.target_id,
            backend=self.backend_name,
            reason=request.reason,
            evidence={
                "control_type": getattr(control.element_info, "control_type", "unknown"),
                "observation_id": self._observation.observation_id,
            },
        )

    def close(self) -> None:
        self._controls.clear()
        self._node_specs.clear()
        self._closed = True
        self._observation = None
