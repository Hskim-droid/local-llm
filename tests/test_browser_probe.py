import json
import unittest
from pathlib import Path

import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
import browser_probe  # noqa: E402
from surface_adapter import ActionRequest  # noqa: E402


@unittest.skipUnless(browser_probe.IMPORT_ERROR is None, "browser probe dependencies are not installed")
class BrowserProbeTests(unittest.TestCase):
    def test_adapter_can_execute_a_node_id_from_its_observation(self):
        with browser_probe.sync_playwright() as playwright:
            browser = playwright.chromium.launch(headless=True)
            try:
                page = browser.new_page()
                page.goto((ROOT / "fixtures" / "qms_variant_1.html").resolve().as_uri(), wait_until="load")
                adapter = browser_probe.PlaywrightAriaAdapter(page)
                observation = adapter.observe()
                node = next(node for node in observation.nodes if node.role == "button")
                receipt = adapter.execute(
                    ActionRequest(
                        action="click",
                        target_id=node.node_id,
                        reason="test navigation",
                        observation_id=observation.observation_id,
                    ),
                )
                self.assertTrue(receipt.accepted)
                self.assertTrue(page.get_by_role("menuitem", name="Quality open issues").is_visible())
            finally:
                browser.close()

    def test_browser_rejects_a_node_that_changed_since_observation(self):
        with browser_probe.sync_playwright() as playwright:
            browser = playwright.chromium.launch(headless=True)
            try:
                page = browser.new_page()
                page.goto((ROOT / "fixtures" / "qms_variant_1.html").resolve().as_uri(), wait_until="load")
                adapter = browser_probe.PlaywrightAriaAdapter(page)
                observation = adapter.observe()
                node = next(node for node in observation.nodes if node.role == "button")
                page.evaluate(
                    """() => {
                        const button = document.createElement('button');
                        button.setAttribute('aria-label', 'Injected navigation');
                        document.querySelector('header').prepend(button);
                    }"""
                )
                with self.assertRaises(browser_probe.ProbeError):
                    adapter.execute(ActionRequest("click", node.node_id, "stale target", observation.observation_id))
            finally:
                browser.close()

    def test_browser_rejects_an_action_pinned_to_an_old_observation(self):
        with browser_probe.sync_playwright() as playwright:
            browser = playwright.chromium.launch(headless=True)
            try:
                page = browser.new_page()
                page.goto((ROOT / "fixtures" / "qms_variant_1.html").resolve().as_uri(), wait_until="load")
                adapter = browser_probe.PlaywrightAriaAdapter(page)
                first = adapter.observe()
                node = next(node for node in first.nodes if node.role == "button")
                page.evaluate(
                    """() => {
                        const button = document.createElement('button');
                        button.setAttribute('aria-label', 'New navigation state');
                        document.querySelector('header').append(button);
                    }"""
                )
                adapter.observe()
                with self.assertRaises(browser_probe.ProbeError):
                    adapter.execute(
                        ActionRequest(
                            "click",
                            node.node_id,
                            "old observation",
                            observation_id=first.observation_id,
                        )
                    )
            finally:
                browser.close()

    def test_browser_rejects_replaced_node_with_same_observation_hash(self):
        with browser_probe.sync_playwright() as playwright:
            browser = playwright.chromium.launch(headless=True)
            try:
                page = browser.new_page()
                page.goto((ROOT / "fixtures" / "qms_variant_1.html").resolve().as_uri(), wait_until="load")
                adapter = browser_probe.PlaywrightAriaAdapter(page)
                first = adapter.observe()
                node = next(node for node in first.nodes if node.role == "button")
                page.evaluate(
                    """() => {
                        const current = document.querySelector('header button');
                        current.replaceWith(current.cloneNode(true));
                    }"""
                )
                second = adapter.observe()
                self.assertEqual(first.observation_hash, second.observation_hash)
                self.assertNotEqual(first.observation_id, second.observation_id)
                with self.assertRaises(browser_probe.ProbeError):
                    adapter.execute(ActionRequest("click", node.node_id, "old generation", first.observation_id))
            finally:
                browser.close()

    def test_one_task_contract_handles_three_menu_and_table_variants(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        expected_ids = ["QMS-1001", "QMS-1002", "QMS-1003"]
        for number in (1, 2, 3):
            with self.subTest(variant=number):
                result = browser_probe.run_task(ROOT / "fixtures" / f"qms_variant_{number}.html", task)
                self.assertEqual([record["issue_id"] for record in result["records"]], expected_ids)
                self.assertEqual(result["field_mapping"]["issue_id"], 0 if number != 2 else 1)
                self.assertTrue(all(check["passed"] for check in result["checks"]))
                self.assertGreaterEqual(len(result["actions"]), 2 if number < 3 else 4)
                self.assertIn("read_table", result["observation_after"]["capabilities"])
                self.assertEqual(result["observation_before"]["schema_version"], 1)
                self.assertEqual(result["observation_after"]["backend"], "playwright-aria")
                self.assertTrue(result["actions"][0]["action_id"].startswith("act-"))
                self.assertEqual(result["actions"][0]["action"], "click")
                self.assertIn("target_id", result["actions"][0])

    def test_ambiguous_navigation_stops_without_clicking(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        html = (ROOT / "fixtures" / "qms_variant_1.html").read_text(encoding="utf-8")
        html = html.replace(
            "<button role=\"menuitem\" onclick=\"showIssues()\">Quality open issues</button>",
            "<button role=\"menuitem\" onclick=\"showIssues()\">Quality open issues</button>"
            "<button role=\"menuitem\" onclick=\"showIssues()\">Quality open issues</button>",
        )
        path = ROOT / "fixtures" / ".tmp-ambiguous-qms.html"
        path.write_text(html, encoding="utf-8")
        try:
            with self.assertRaises(browser_probe.ProbeError):
                browser_probe.run_task(path, task)
        finally:
            path.unlink()

    def test_unpublished_menu_label_variant_uses_the_same_contract(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        html = (ROOT / "fixtures" / "qms_variant_1.html").read_text(encoding="utf-8")
        html = html.replace("Quality open issues", "Quality exceptions — open issues")
        path = ROOT / "fixtures" / ".tmp-unpublished-qms.html"
        path.write_text(html, encoding="utf-8")
        try:
            result = browser_probe.run_task(path, task)
            self.assertEqual([record["issue_id"] for record in result["records"]], ["QMS-1001", "QMS-1002", "QMS-1003"])
        finally:
            path.unlink()

    def test_matching_column_aliases_must_be_one_to_one(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        html = (ROOT / "fixtures" / "qms_variant_1.html").read_text(encoding="utf-8")
        html = html.replace("<th>Title</th>", "").replace("<td>Incoming inspection hold</td>", "")
        html = html.replace("<td>Calibration evidence missing</td>", "").replace("<td>Supplier corrective action overdue</td>", "")
        path = ROOT / "fixtures" / ".tmp-colliding-columns.html"
        path.write_text(html, encoding="utf-8")
        try:
            with self.assertRaises(browser_probe.ProbeError):
                browser_probe.run_task(path, task)
        finally:
            path.unlink()

    def test_forbidden_menu_item_is_rejected_before_click(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        html = (ROOT / "fixtures" / "qms_variant_1.html").read_text(encoding="utf-8")
        html = html.replace(
            "<button role=\"menuitem\" onclick=\"showIssues()\">Quality open issues</button>",
            "<button role=\"menuitem\" onclick=\"document.body.dataset.mutated='yes'\">Delete quality issue</button>",
        )
        path = ROOT / "fixtures" / ".tmp-forbidden-qms.html"
        path.write_text(html, encoding="utf-8")
        try:
            with self.assertRaises(browser_probe.ProbeError):
                browser_probe.run_task(path, task)
        finally:
            path.unlink()

    def test_table_identity_prevents_selecting_a_completed_same_schema_table(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        html = (ROOT / "fixtures" / "qms_variant_1.html").read_text(encoding="utf-8")
        completed = """
        <table aria-label="Completed quality issues">
          <thead><tr><th>Issue ID</th><th>Title</th><th>Status</th><th>Owner</th><th>Updated</th></tr></thead>
          <tbody><tr><td>QMS-9999</td><td>Closed issue</td><td>Closed</td><td>J. Kim</td><td>2026-01-01</td></tr></tbody>
        </table>
        """
        html = html.replace("<main id=\"issues\" hidden>", completed + "<main id=\"issues\" hidden>")
        path = ROOT / "fixtures" / ".tmp-completed-qms.html"
        path.write_text(html, encoding="utf-8")
        try:
            result = browser_probe.run_task(path, task)
            self.assertNotIn("QMS-9999", [record["issue_id"] for record in result["records"]])
        finally:
            path.unlink()

    def test_task_requires_table_identity_and_forbidden_action_policy(self):
        task = json.loads((ROOT / "fixtures" / "qms_task.json").read_text(encoding="utf-8"))
        for key in ("table_terms", "forbidden_terms"):
            with self.subTest(key=key):
                invalid = dict(task)
                invalid[key] = []
                with self.assertRaises(browser_probe.ProbeError):
                    browser_probe.run_task(ROOT / "fixtures" / "qms_variant_1.html", invalid)


if __name__ == "__main__":
    unittest.main()
