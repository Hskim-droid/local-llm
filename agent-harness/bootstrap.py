#!/usr/bin/env python3
"""Detect a host and prepare a bounded local adapter environment.

The default commands only inspect the host or print a plan. Installation is
explicit, network-gated, and limited to the repository's declared Python
requirements, Playwright's Chromium binary, and optional native bindings.
This module never discovers a desktop-wide root, credentials, cookies, or an
ERP window on behalf of a caller.
"""

from __future__ import annotations

import argparse
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
import importlib.util
import json
from pathlib import Path
import platform
import shutil
import subprocess
import sys
from typing import Any


class BootstrapError(RuntimeError):
    """The host cannot satisfy a bounded bootstrap operation."""


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def repo_root() -> Path:
    return Path(__file__).resolve().parent


def module_available(name: str) -> bool:
    try:
        return importlib.util.find_spec(name) is not None
    except (ImportError, ValueError):
        return False


def command_available(name: str) -> bool:
    return shutil.which(name) is not None


def browser_cache_available() -> bool:
    # Playwright knows the revision required by the installed Python package;
    # use that executable path rather than treating any old cache directory as
    # compatible. This starts only the local Playwright driver and never a
    # browser page.
    if module_available("playwright"):
        try:
            from playwright.sync_api import sync_playwright

            playwright = sync_playwright().start()
            try:
                if Path(playwright.chromium.executable_path).is_file():
                    return True
            finally:
                playwright.stop()
        except (ImportError, OSError, RuntimeError):
            pass
    return False


@dataclass(frozen=True)
class EnvironmentReport:
    schema_version: int
    captured_at: str
    os_name: str
    platform: str
    machine: str
    python_version: str
    package_managers: dict[str, bool]
    modules: dict[str, bool]
    local_model_commands: dict[str, bool]
    browser_channels: dict[str, bool]
    browser_cache: bool
    capabilities: dict[str, bool]

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class PlanStep:
    step_id: str
    purpose: str
    command: tuple[str, ...]
    network: bool
    status: str

    def to_dict(self) -> dict[str, Any]:
        result = asdict(self)
        result["command"] = list(self.command)
        return result


def collect_report() -> EnvironmentReport:
    os_name = platform.system().lower()
    modules = {
        "playwright": module_available("playwright"),
        "docx": module_available("docx"),
        "openpyxl": module_available("openpyxl"),
        "pptx": module_available("pptx"),
        "Quartz": module_available("Quartz"),
        "ApplicationServices": module_available("ApplicationServices"),
        "pywinauto": module_available("pywinauto"),
        "atspi": module_available("pyatspi"),
    }
    local_model_commands = {
        "ollama": command_available("ollama"),
        "llama_cli": command_available("llama-cli"),
        "llama_server": command_available("llama-server"),
    }
    browser_channels = {
        "chromium": command_available("chromium") or command_available("chromium-browser"),
        "chrome": command_available("google-chrome") or command_available("chrome"),
        "edge": command_available("msedge") or command_available("microsoft-edge"),
        "firefox": command_available("firefox"),
        "webkit": False,
    }
    package_managers = {
        "python": command_available("python3") or command_available("python"),
        "uv": command_available("uv"),
        "brew": command_available("brew"),
        "winget": command_available("winget"),
    }
    capabilities = {
        "browser_playwright": modules["playwright"],
        "macos_ax_binding": os_name == "darwin" and (modules["Quartz"] or modules["ApplicationServices"]),
        "windows_uia_binding": os_name == "windows" and modules["pywinauto"],
        "linux_atspi_binding": os_name == "linux" and modules["atspi"],
        "docx_renderer": modules["docx"],
        "xlsx_renderer": modules["openpyxl"],
        "pptx_renderer": modules["pptx"],
        "local_model_engine": any(local_model_commands.values()),
    }
    return EnvironmentReport(
        schema_version=1,
        captured_at=utc_now(),
        os_name=os_name,
        platform=platform.platform(aliased=True, terse=True),
        machine=platform.machine() or "unknown",
        python_version=platform.python_version(),
        package_managers=package_managers,
        modules=modules,
        local_model_commands=local_model_commands,
        browser_channels=browser_channels,
        browser_cache=browser_cache_available(),
        capabilities=capabilities,
    )


def _python_command() -> str:
    return sys.executable


def build_plan(report: EnvironmentReport, profile: str, root: Path | None = None) -> list[PlanStep]:
    if profile not in {"browser", "native", "local-model", "all"}:
        raise BootstrapError("profile must be browser, native, local-model, or all")
    root = root or repo_root()
    steps: list[PlanStep] = []
    needs_browser = profile in {"browser", "all"}
    needs_native = profile in {"native", "all"}
    needs_local_model = profile in {"local-model", "all"}

    if needs_browser and not all(report.modules[name] for name in ("playwright", "docx", "openpyxl", "pptx")):
        steps.append(
            PlanStep(
                "python-fixture-dependencies",
                "install the declared fixture Python requirements",
                (_python_command(), "-m", "pip", "install", "-r", str(root / "requirements-fixture.txt")),
                True,
                "needed",
            )
        )
    # The plan is ordered: requirements are installed before the browser
    # binary, so a first-run report can safely include both steps.
    if needs_browser and not report.browser_cache:
        steps.append(
            PlanStep(
                "playwright-chromium",
                "install the Playwright Chromium browser for the local browser adapter",
                (_python_command(), "-m", "playwright", "install", "chromium"),
                True,
                "needed",
            )
        )
    if needs_native and report.os_name == "darwin" and not report.capabilities["macos_ax_binding"]:
        steps.append(
            PlanStep(
                "macos-ax-binding",
                "install the optional macOS AXUIElement Python binding",
                (_python_command(), "-m", "pip", "install", "pyobjc-framework-Quartz"),
                True,
                "needed",
            )
        )
    if needs_native and report.os_name == "windows" and not report.capabilities["windows_uia_binding"]:
        steps.append(
            PlanStep(
                "windows-uia-binding",
                "install the optional Windows UI Automation binding",
                (_python_command(), "-m", "pip", "install", "pywinauto"),
                True,
                "needed",
            )
        )
    if needs_native and report.os_name == "linux":
        steps.append(
            PlanStep(
                "linux-native-adapter",
                "Linux AT-SPI adapter is not bundled; select a reviewed adapter explicitly",
                (),
                False,
                "manual-review",
            )
        )
    if needs_local_model and not report.capabilities["local_model_engine"]:
        steps.append(
            PlanStep(
                "local-model-engine",
                "select and install a reviewed local model engine and model profile explicitly",
                (),
                False,
                "manual-review",
            )
        )
    if not steps:
        steps.append(PlanStep("no-op", "all selected capabilities are present", (), False, "ready"))
    return steps


def _expected_command(step_id: str) -> tuple[str, ...] | None:
    python = _python_command()
    requirements = str(repo_root() / "requirements-fixture.txt")
    return {
        "python-fixture-dependencies": (python, "-m", "pip", "install", "-r", requirements),
        "playwright-chromium": (python, "-m", "playwright", "install", "chromium"),
        "macos-ax-binding": (python, "-m", "pip", "install", "pyobjc-framework-Quartz"),
        "windows-uia-binding": (python, "-m", "pip", "install", "pywinauto"),
    }.get(step_id)


_NON_EXECUTING_STEP_IDS = {
    "linux-native-adapter",
    "local-model-engine",
    "no-op",
}


def apply_plan(steps: list[PlanStep], *, allow_network: bool) -> list[dict[str, Any]]:
    """Apply only a plan shape produced by this module.

    The network flag is an approval gate, not a command authorization. Validate
    the step ID and exact argument tuple before invoking subprocess.
    """
    if any(step.network for step in steps) and not allow_network:
        raise BootstrapError("installation needs network access; rerun apply with --allow-network")
    results: list[dict[str, Any]] = []
    for step in steps:
        expected_command = _expected_command(step.step_id)
        if expected_command is None and step.step_id not in _NON_EXECUTING_STEP_IDS:
            raise BootstrapError(f"refusing an unrecognized bootstrap step: {step.step_id}")
        if expected_command is not None and tuple(step.command) != expected_command:
            raise BootstrapError(f"refusing modified command arguments: {step.step_id}")
        if expected_command is None and step.command:
            raise BootstrapError(f"refusing a command on a non-executing step: {step.step_id}")
        if not step.command:
            results.append({"step_id": step.step_id, "status": step.status})
            continue
        if not step.network:
            raise BootstrapError(f"refusing an unrecognized non-network command: {step.step_id}")
        completed = subprocess.run(step.command, cwd=repo_root(), check=False)
        result = {"step_id": step.step_id, "status": "applied" if completed.returncode == 0 else "failed", "returncode": completed.returncode}
        results.append(result)
        if completed.returncode != 0:
            raise BootstrapError(f"bootstrap step failed: {step.step_id}")
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Detect and prepare bounded local adapter capabilities")
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("report", help="read host capabilities without changing the machine")
    plan = sub.add_parser("plan", help="print the install plan without changing the machine")
    plan.add_argument("--profile", choices=("browser", "native", "local-model", "all"), default="browser")
    apply = sub.add_parser("apply", help="run the allowlisted install plan")
    apply.add_argument("--profile", choices=("browser", "native", "local-model", "all"), default="browser")
    apply.add_argument("--allow-network", action="store_true", help="explicitly permit package/browser downloads")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        report = collect_report()
        if args.command == "report":
            print(json.dumps(report.to_dict(), ensure_ascii=False, indent=2))
            return 0
        steps = build_plan(report, args.profile)
        if args.command == "plan":
            print(json.dumps({"report": report.to_dict(), "steps": [step.to_dict() for step in steps]}, ensure_ascii=False, indent=2))
            return 0
        results = apply_plan(steps, allow_network=args.allow_network)
        print(json.dumps({"report": report.to_dict(), "results": results}, ensure_ascii=False, indent=2))
        return 0
    except (BootstrapError, OSError, ValueError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
