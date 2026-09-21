import unittest

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from surface_adapter import ActionReceipt, UiNode, UiObservation  # noqa: E402


class SurfaceAdapterContractTests(unittest.TestCase):
    def test_observation_is_json_ready_and_separates_generation_from_content(self):
        node = UiNode(
            node_id="test:button:0",
            role="button",
            name="Open",
            state={"enabled": True},
            actions=("click",),
        )
        first = UiObservation(
            backend="test",
            surface="browser",
            title="Fixture",
            url="https://example.test",
            nodes=(node,),
            capabilities=("observe_tree", "navigate"),
        )
        second = UiObservation(
            backend="test",
            surface="browser",
            title="Fixture",
            url="https://example.test",
            nodes=(node,),
            capabilities=("observe_tree", "navigate"),
        )
        payload = first.to_dict()
        self.assertNotEqual(first.observation_id, second.observation_id)
        self.assertEqual(first.observation_hash, second.observation_hash)
        self.assertEqual(payload["schema_version"], 1)
        self.assertEqual(len(payload["observation_id"]), 36)
        self.assertEqual(payload["nodes"][0]["actions"], ["click"])
        self.assertEqual(payload["nodes"][0]["source_ref"], {})

    def test_action_receipt_is_explicit_evidence(self):
        receipt = ActionReceipt(
            action_id="act-test",
            action="click",
            target_id="test:button:0",
            backend="test",
            reason="open navigation",
        )
        self.assertTrue(receipt.to_dict()["accepted"])
        self.assertEqual(receipt.to_dict()["target_id"], "test:button:0")
