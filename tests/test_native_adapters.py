import unittest

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from native_adapters import (  # noqa: E402
    MacAXAdapter,
    NativeAdapterUnavailable,
    WindowsUIAAdapter,
    backend_status,
)
from surface_adapter import ActionRequest  # noqa: E402


class FakeAXElement:
    def __init__(self, role, title="", *, actions=(), children=(), value=None, enabled=True):
        self.attrs = {
            "AXRole": role,
            "AXTitle": title,
            "AXActions": list(actions),
            "AXChildren": list(children),
            "AXValue": value,
            "AXEnabled": enabled,
        }
        self.performed = []


class FakeAXApi:
    def AXUIElementCopyAttributeValue(self, element, attribute, _unused):
        return 0, element.attrs.get(attribute)

    def AXUIElementPerformAction(self, element, action):
        element.performed.append(action)
        return 0

    def AXUIElementCopyActionNames(self, element, _unused):
        return 0, element.attrs.get("AXActions", [])


class PermissionAXApi(FakeAXApi):
    def AXUIElementCopyAttributeValue(self, _element, _attribute, _unused):
        return -25211, None


class BrokenRoleAXApi(FakeAXApi):
    def AXUIElementCopyAttributeValue(self, _element, attribute, _unused):
        if attribute == "AXRole":
            return -25204, None
        return 0, None


class PartialFailureAXApi(FakeAXApi):
    def AXUIElementCopyAttributeValue(self, element, attribute, unused):
        if getattr(element, "broken", False) and attribute == "AXRole":
            return -25204, None
        return super().AXUIElementCopyAttributeValue(element, attribute, unused)


class FakeUIAInfo:
    def __init__(self, control_type, runtime_id):
        self.control_type = control_type
        self.runtime_id = runtime_id


class FakeUIAControl:
    def __init__(self, control_type, runtime_id, name, *, children=(), enabled=True):
        self.element_info = FakeUIAInfo(control_type, runtime_id)
        self.name = name
        self.children = list(children)
        self.enabled = enabled
        self.clicked = False

    def descendants(self):
        return self.children

    def window_text(self):
        return self.name

    def is_enabled(self):
        return self.enabled

    def invoke(self):
        self.clicked = True


class BrokenUIAControl(FakeUIAControl):
    @property
    def element_info(self):
        raise RuntimeError("UIA control disconnected")

    @element_info.setter
    def element_info(self, _value):
        pass


class NativeAdapterTests(unittest.TestCase):
    def test_backend_status_is_safe_without_native_bindings(self):
        statuses = {status.name: status for status in backend_status()}
        self.assertIn("playwright-aria", statuses)
        self.assertIn("macos-ax", statuses)
        self.assertIn("windows-uia", statuses)
        self.assertIn("linux-atspi", statuses)
        self.assertTrue(all(status.reason for status in statuses.values()))

    def test_macos_ax_adapter_normalizes_tree_and_executes_press(self):
        button = FakeAXElement("AXButton", "Open", actions=("AXPress",))
        root = FakeAXElement("AXApplication", "Fixture", children=(button,))
        adapter = MacAXAdapter(root=root, api=FakeAXApi(), title="Fixture")

        observation = adapter.observe()
        node = next(node for node in observation.nodes if node.name == "Open")
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", node.node_id, "stale", observation_id="old-observation"))
        self.assertEqual(button.performed, [])
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", node.node_id, "missing observation", observation_id=""))
        receipt = adapter.execute(ActionRequest("click", node.node_id, "open menu", observation.observation_id))

        self.assertEqual(receipt.backend, "macos-ax")
        self.assertEqual(button.performed, ["AXPress"])
        self.assertEqual(receipt.evidence["observation_id"], observation.observation_id)
        self.assertIn("native_accessibility", observation.capabilities)

    def test_macos_permission_failure_is_not_downgraded_to_empty_tree(self):
        adapter = MacAXAdapter(root=FakeAXElement("AXApplication"), api=PermissionAXApi())
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.observe()

        broken = MacAXAdapter(root=FakeAXElement("AXApplication"), api=BrokenRoleAXApi())
        with self.assertRaises(NativeAdapterUnavailable):
            broken.observe()

    def test_macos_failed_observation_discards_nodes_collected_before_error(self):
        good = FakeAXElement("AXButton", "Good", actions=("AXPress",))
        broken = FakeAXElement("AXButton", "Broken")
        broken.broken = True
        root = FakeAXElement("AXApplication", "Fixture", children=(good, broken))
        adapter = MacAXAdapter(root=root, api=PartialFailureAXApi())

        with self.assertRaises(NativeAdapterUnavailable):
            adapter.observe()
        self.assertEqual(adapter._elements, {})
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", "macos-ax:0.0", "stale after failed observation", "failed-observation"))
        self.assertEqual(good.performed, [])

    def test_macos_capability_check_does_not_remap_target_after_tree_reorder(self):
        read = FakeAXElement("AXButton", "Read", actions=("AXPress",))
        delete = FakeAXElement("AXButton", "Delete", actions=("AXPress",))
        root = FakeAXElement("AXApplication", "Fixture", children=(read, delete))
        adapter = MacAXAdapter(root=root, api=FakeAXApi())
        observation = adapter.observe()
        root.attrs["AXChildren"] = [delete, read]

        adapter.execute(
            ActionRequest(
                "click",
                "macos-ax:0.0",
                "read",
                observation.observation_id,
                expected_capabilities=("navigate",),
            )
        )
        self.assertEqual(read.performed, ["AXPress"])
        self.assertEqual(delete.performed, [])

    def test_macos_rejects_replaced_element_with_same_observation_hash(self):
        old_button = FakeAXElement("AXButton", "Open", actions=("AXPress",))
        root = FakeAXElement("AXApplication", "Fixture", children=(old_button,))
        adapter = MacAXAdapter(root=root, api=FakeAXApi())
        first = adapter.observe()
        replacement = FakeAXElement("AXButton", "Open", actions=("AXPress",))
        root.attrs["AXChildren"] = [replacement]
        second = adapter.observe()

        self.assertEqual(first.observation_hash, second.observation_hash)
        self.assertNotEqual(first.observation_id, second.observation_id)
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", "macos-ax:0.0", "old generation", first.observation_id))
        self.assertEqual(replacement.performed, [])
        adapter.execute(ActionRequest("click", "macos-ax:0.0", "current generation", second.observation_id))
        self.assertEqual(replacement.performed, ["AXPress"])

    def test_macos_rejects_unobserved_target_disabled_target_and_partial_tree(self):
        button = FakeAXElement("AXButton", "Open", actions=("AXPress",))
        root = FakeAXElement("AXApplication", "Fixture", children=(button,))
        adapter = MacAXAdapter(root=root, api=FakeAXApi())
        observation = adapter.observe()
        node = next(node for node in observation.nodes if node.name == "Open")

        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(
                ActionRequest("click", node.node_id, "outside", observation.observation_id),
                target=FakeAXElement("AXButton"),
            )
        self.assertEqual(button.performed, [])

        adapter.close()
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", node.node_id, "after close", observation.observation_id))

        limited = MacAXAdapter(root=root, api=FakeAXApi(), max_depth=0)
        partial = limited.observe()
        self.assertIn("partial_tree", partial.capabilities)
        with self.assertRaises(NativeAdapterUnavailable):
            limited.execute(ActionRequest("click", "macos-ax:0", "partial", partial.observation_id))

    def test_macos_rejects_disabled_or_unsupported_action_before_native_call(self):
        button = FakeAXElement("AXButton", "Disabled", actions=("AXPress",), enabled=False)
        root = FakeAXElement("AXApplication", "Fixture", children=(button,))
        adapter = MacAXAdapter(root=root, api=FakeAXApi())
        observation = adapter.observe()
        node = next(node for node in observation.nodes if node.name == "Disabled")
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", node.node_id, "disabled", observation.observation_id))
        self.assertEqual(button.performed, [])

        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(
                ActionRequest(
                    "click",
                    node.node_id,
                    "missing capability",
                    observation.observation_id,
                    expected_capabilities=("read_table",),
                )
            )
        self.assertEqual(button.performed, [])

    def test_windows_uia_adapter_normalizes_tree_and_clicks(self):
        button = FakeUIAControl("Button", (1, 2), "Open")
        root = FakeUIAControl("Window", (1,), "Fixture", children=(button,))
        adapter = WindowsUIAAdapter(root=root, api=object())

        observation = adapter.observe()
        node = next(node for node in observation.nodes if node.name == "Open")
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", node.node_id, "missing observation", ""))
        receipt = adapter.execute(ActionRequest("click", node.node_id, "open menu", observation.observation_id))

        self.assertEqual(receipt.backend, "windows-uia")
        self.assertTrue(button.clicked)
        self.assertIn("navigate", observation.capabilities)

    def test_windows_rejects_external_target_and_descendant_failure(self):
        button = FakeUIAControl("Button", (1, 2), "Open")
        root = FakeUIAControl("Window", (1,), "Fixture", children=(button,))
        adapter = WindowsUIAAdapter(root=root, api=object())
        observation = adapter.observe()
        node = next(node for node in observation.nodes if node.name == "Open")
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(
                ActionRequest("click", node.node_id, "outside", observation.observation_id),
                target=FakeUIAControl("Button", (9,), "Other"),
            )
        self.assertFalse(button.clicked)
        adapter.close()
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", node.node_id, "after close", observation.observation_id))

        class FailingRoot(FakeUIAControl):
            def descendants(self):
                raise RuntimeError("UIA disconnected")

        failing = WindowsUIAAdapter(root=FailingRoot("Window", (2,), "Broken"), api=object())
        with self.assertRaises(NativeAdapterUnavailable):
            failing.observe()
        self.assertEqual(failing._controls, {})

    def test_windows_failed_observation_discards_controls_collected_before_error(self):
        good = FakeUIAControl("Button", (1, 2), "Good")
        broken = BrokenUIAControl("Button", (1, 3), "Broken")
        root = FakeUIAControl("Window", (1,), "Fixture", children=(good, broken))
        adapter = WindowsUIAAdapter(root=root, api=object())

        with self.assertRaises(NativeAdapterUnavailable):
            adapter.observe()
        self.assertEqual(adapter._controls, {})
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", "windows-uia:1-2", "stale after failed observation", "failed-observation"))
        self.assertFalse(good.clicked)

    def test_windows_rejects_replaced_control_with_same_observation_hash(self):
        old_button = FakeUIAControl("Button", (1, 2), "Open")
        root = FakeUIAControl("Window", (1,), "Fixture", children=(old_button,))
        adapter = WindowsUIAAdapter(root=root, api=object())
        first = adapter.observe()
        replacement = FakeUIAControl("Button", (1, 2), "Open")
        root.children = [replacement]
        second = adapter.observe()

        self.assertEqual(first.observation_hash, second.observation_hash)
        self.assertNotEqual(first.observation_id, second.observation_id)
        with self.assertRaises(NativeAdapterUnavailable):
            adapter.execute(ActionRequest("click", "windows-uia:1-2", "old generation", first.observation_id))
        self.assertFalse(replacement.clicked)
        adapter.execute(ActionRequest("click", "windows-uia:1-2", "current generation", second.observation_id))
        self.assertTrue(replacement.clicked)

    def test_native_adapters_require_explicit_windows_root(self):
        with self.assertRaises(NativeAdapterUnavailable):
            WindowsUIAAdapter()

    def test_macos_adapter_requires_explicit_application_scope(self):
        with self.assertRaises(NativeAdapterUnavailable):
            MacAXAdapter(api=FakeAXApi())


if __name__ == "__main__":
    unittest.main()
