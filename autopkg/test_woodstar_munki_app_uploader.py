import sys
import types
import unittest
from pathlib import Path


class StubProcessor:
    def __init__(self):
        self.env = {}
        self.output_messages = []

    def output(self, message):
        self.output_messages.append(message)

    def execute_shell(self):
        raise NotImplementedError


class StubProcessorError(Exception):
    pass


autopkglib = types.ModuleType("autopkglib")
autopkglib.Processor = StubProcessor
autopkglib.ProcessorError = StubProcessorError
sys.modules["autopkglib"] = autopkglib

AUTOPKG_DIR = Path(__file__).resolve().parent
if str(AUTOPKG_DIR) not in sys.path:
    sys.path.insert(0, str(AUTOPKG_DIR))

from WoodstarMunkiAppUploader import WoodstarMunkiAppUploader  # noqa: E402
from autopkglib import ProcessorError  # noqa: E402


class FakeClient:
    def __init__(self):
        self.labels = [
            {"id": 10, "name": "Staff Macs"},
            {"id": 11, "name": "Lab Macs"},
            {"id": 20, "name": "Retired Macs"},
        ]

    def get(self, path, query=None):
        if path != "/api/labels":
            raise AssertionError(f"unexpected GET {path}")
        q = (query or {}).get("q")
        items = [label for label in self.labels if q is None or q in label["name"]]
        return {"items": items, "count": len(items)}


def processor(env=None):
    proc = WoodstarMunkiAppUploader()
    proc.env = env or {}
    return proc


class WoodstarMunkiAppUploaderTargetTests(unittest.TestCase):
    def test_preserves_existing_targets_without_env_targets(self):
        existing_targets = {
            "include": [
                {
                    "label_id": 10,
                    "package": {"strategy": "latest"},
                    "state": "managed_install",
                    "featured": False,
                }
            ],
            "exclude": [{"label_id": 20}],
        }

        got = processor().target_body(FakeClient(), {"targets": existing_targets})

        self.assertEqual(got, existing_targets)

    def test_converts_label_names_and_package_selectors(self):
        proc = processor(
            {
                "targets": {
                    "include": [
                        {
                            "label_name": "Staff Macs",
                            "package": {"strategy": "latest"},
                            "state": "managed_install",
                            "featured": False,
                        },
                        {
                            "label_id": 11,
                            "package": {"strategy": "specific", "package_id": "123"},
                            "state": "managed_update",
                            "featured": "false",
                        },
                    ],
                    "exclude": [{"label_name": "Retired Macs"}],
                }
            }
        )

        got = proc.target_body(FakeClient(), None)

        self.assertEqual(
            got,
            {
                "include": [
                    {
                        "label_id": 10,
                        "package": {"strategy": "latest"},
                        "state": "managed_install",
                        "featured": False,
                    },
                    {
                        "label_id": 11,
                        "package": {"strategy": "specific", "package_id": 123},
                        "state": "managed_update",
                        "featured": False,
                    },
                ],
                "exclude": [{"label_id": 20}],
            },
        )

    def test_rejects_duplicate_exclude_labels(self):
        proc = processor(
            {
                "targets": {
                    "include": [],
                    "exclude": [{"label_id": 20}, {"label_name": "Retired Macs"}],
                }
            }
        )

        with self.assertRaisesRegex(ProcessorError, "duplicate"):
            proc.target_body(FakeClient(), None)

    def test_rejects_include_priority(self):
        proc = processor(
            {
                "targets": {
                    "include": [{"label_id": 10, "priority": 1}],
                    "exclude": [],
                }
            }
        )

        with self.assertRaisesRegex(ProcessorError, "priority"):
            proc.target_body(FakeClient(), None)


if __name__ == "__main__":
    unittest.main()
