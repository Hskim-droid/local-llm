import unittest
from unittest import mock

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from bootstrap import (  # noqa: E402
    BootstrapError,
    EnvironmentReport,
    PlanStep,
    apply_plan,
    build_plan,
    collect_report,
)


def report_for(**overrides):
    values = {
        "schema_version": 1,
        "captured_at": "2026-09-21T00:00:00+00:00",
        "os_name": "darwin",
        "platform": "Darwin-test",
        "machine": "arm64",
        "python_version": "3.13.0",
        "package_managers": {"python": True, "uv": False, "brew": True, "winget": False},
        "modules": {"playwright": False, "docx": False, "openpyxl": False, "pptx": False, "Quartz": False, "ApplicationServices": False, "pywinauto": False, "atspi": False},
        "local_model_commands": {"ollama": False, "llama_cli": False, "llama_server": False},
        "browser_channels": {"chromium": False, "chrome": True, "edge": False, "firefox": False, "webkit": False},
        "browser_cache": False,
        "capabilities": {"browser_playwright": False, "macos_ax_binding": False, "windows_uia_binding": False, "linux_atspi_binding": False, "docx_renderer": False, "xlsx_renderer": False, "pptx_renderer": False, "local_model_engine": False},
    }
    values.update(overrides)
    return EnvironmentReport(**values)


class BootstrapTests(unittest.TestCase):
    def test_report_contains_no_host_identity_and_expected_capabilities(self):
        report = collect_report().to_dict()
        self.assertEqual(report["schema_version"], 1)
        self.assertIn("os_name", report)
        self.assertIn("capabilities", report)
        self.assertIn("local_model_commands", report)
        self.assertNotIn("hostname", report)
        self.assertNotIn("username", report)

    def test_browser_plan_is_explicit_and_allowlisted(self):
        steps = build_plan(report_for(), "browser", ROOT)
        step_ids = {step.step_id for step in steps}
        self.assertIn("python-fixture-dependencies", step_ids)
        self.assertIn("playwright-chromium", step_ids)
        commands = [part for step in steps for part in step.command]
        self.assertNotIn("bash", commands)
        self.assertTrue(all(step.network for step in steps))

    def test_native_plan_selects_macos_binding_without_windows_commands(self):
        steps = build_plan(report_for(), "native", ROOT)
        self.assertEqual([step.step_id for step in steps], ["macos-ax-binding"])
        self.assertIn("pyobjc-framework-Quartz", steps[0].command)
        self.assertNotIn("pywinauto", steps[0].command)

    def test_linux_native_plan_requires_manual_review(self):
        report = report_for(
            os_name="linux",
            modules={"playwright": True, "docx": True, "openpyxl": True, "pptx": True, "Quartz": False, "ApplicationServices": False, "pywinauto": False, "atspi": False},
            capabilities={"browser_playwright": True, "macos_ax_binding": False, "windows_uia_binding": False, "linux_atspi_binding": False, "docx_renderer": True, "xlsx_renderer": True, "pptx_renderer": True, "local_model_engine": False},
        )
        steps = build_plan(report, "native", ROOT)
        self.assertEqual(steps[0].status, "manual-review")
        self.assertEqual(steps[0].command, ())

    def test_apply_requires_explicit_network_permission(self):
        steps = build_plan(report_for(), "browser", ROOT)
        with self.assertRaises(BootstrapError):
            apply_plan(steps, allow_network=False)

    def test_apply_rejects_an_arbitrary_network_command_before_subprocess(self):
        arbitrary = PlanStep("arbitrary", "test", ("arbitrary-program", "arg"), True, "needed")
        with mock.patch("bootstrap.subprocess.run") as run:
            with self.assertRaises(BootstrapError):
                apply_plan([arbitrary], allow_network=True)
        run.assert_not_called()

    def test_apply_preserves_allowlisted_dependency_then_browser_order(self):
        report = report_for()
        steps = build_plan(report, "browser", ROOT)
        with mock.patch("bootstrap.subprocess.run") as run:
            run.return_value.returncode = 0
            results = apply_plan(steps, allow_network=True)
        self.assertEqual([item["step_id"] for item in results], ["python-fixture-dependencies", "playwright-chromium"])
        self.assertEqual([call.args[0][0:4] for call in run.call_args_list], [
            (sys.executable, "-m", "pip", "install"),
            (sys.executable, "-m", "playwright", "install"),
        ])

    def test_local_model_plan_stops_for_explicit_engine_selection(self):
        steps = build_plan(report_for(), "local-model", ROOT)
        self.assertEqual([step.step_id for step in steps], ["local-model-engine"])
        self.assertEqual(steps[0].status, "manual-review")

    def test_local_model_engine_is_reported_as_ready_when_detected(self):
        report = report_for(
            local_model_commands={"ollama": True, "llama_cli": False, "llama_server": False},
            capabilities={"browser_playwright": False, "macos_ax_binding": False, "windows_uia_binding": False, "linux_atspi_binding": False, "docx_renderer": False, "xlsx_renderer": False, "pptx_renderer": False, "local_model_engine": True},
        )
        steps = build_plan(report, "local-model", ROOT)
        self.assertEqual(steps[0].status, "ready")


if __name__ == "__main__":
    unittest.main()
