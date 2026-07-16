import os
import plistlib
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, os.path.dirname(__file__))
sys.path.insert(0, "/Library/AutoPkg")

from autopkglib import ProcessorError  # noqa: E402
import WoodstarMunkiAppUploader as app_uploader_module  # noqa: E402
from WoodstarMunkiAppUploader import WoodstarMunkiAppUploader  # noqa: E402
from WoodstarLib.Client import local_file_metadata  # noqa: E402
from WoodstarMunkiPackageUploader import (  # noqa: E402
    PackageReferenceResolver,
    WoodstarMunkiPackageUploader,
    default_package_mutation,
    mutation_request_body,
    reference_ids,
)
from WoodstarMunkiRepoImporter import (  # noqa: E402
    WoodstarMunkiRepoImporter,
    import_counts,
    pkginfo_entries,
    pkginfo_key,
)


class PackageReferenceResolverTests(unittest.TestCase):
    def test_resolves_standard_munki_names_and_versions(self):
        resolver = PackageReferenceResolver(
            [{"id": 7, "name": "Dependency"}],
            [
                {
                    "id": 11,
                    "software_id": 7,
                    "software_name": "Dependency",
                    "version": "2.0",
                }
            ],
        )

        self.assertEqual(
            resolver.resolve(
                ["Dependency", "Dependency-2.0", "Dependency--2.0", "Dependency-2.0.0"],
                "requires",
            ),
            [
                {"software_id": 7},
                {"software_id": 7, "package_id": 11},
                {"software_id": 7, "package_id": 11},
                {"software_id": 7, "package_id": 11},
            ],
        )

    def test_rejects_unknown_and_ambiguous_references(self):
        resolver = PackageReferenceResolver(
            [
                {"id": 7, "name": "Dependency"},
                {"id": 8, "name": "Dependency-2.0"},
            ],
            [
                {
                    "id": 11,
                    "software_id": 7,
                    "software_name": "Dependency",
                    "version": "2.0",
                }
            ],
        )

        with self.assertRaises(ProcessorError):
            resolver.resolve(["11"], "requires")
        self.assertEqual(
            resolver.resolve(["Dependency-2.0"], "requires"),
            [{"software_id": 7, "package_id": 11}],
        )

    def test_reference_comparison_includes_software_and_package(self):
        self.assertNotEqual(
            reference_ids([{"software_id": 1}]),
            reference_ids([{"software_id": 2}]),
        )

    def test_request_omits_absent_uninstall_and_restart_actions(self):
        body = default_package_mutation()

        request = mutation_request_body(body)

        self.assertNotIn("uninstall_method", request)
        self.assertNotIn("restart_action", request)

    def test_rejects_packages_with_same_normalized_munki_version(self):
        resolver = PackageReferenceResolver(
            [{"id": 7, "name": "Dependency"}],
            [
                {
                    "id": 11,
                    "software_id": 7,
                    "software_name": "Dependency",
                    "version": "2.0",
                },
                {
                    "id": 12,
                    "software_id": 7,
                    "software_name": "Dependency",
                    "version": "2.0.0",
                },
            ],
        )

        with self.assertRaises(ProcessorError):
            resolver.resolve(["Dependency-2.0.0"], "requires")


class RepoIdentityTests(unittest.TestCase):
    def test_software_identity_uses_name_not_display_name(self):
        uploader = WoodstarMunkiAppUploader()
        uploader.env = {}
        pkginfo = {"name": "InternalName", "display_name": "Friendly Name", "version": "1.0"}

        self.assertEqual(uploader.software_name(pkginfo), "InternalName")
        self.assertEqual(pkginfo_key(pkginfo), ("InternalName", "1.0"))

    def test_duplicate_selection_keeps_distinct_munki_names(self):
        with tempfile.TemporaryDirectory() as repo:
            pkgsinfo = os.path.join(repo, "pkgsinfo")
            os.mkdir(pkgsinfo)
            self.write_pkginfo(pkgsinfo, "first.plist", "First", "Friendly", "1.0")
            self.write_pkginfo(pkgsinfo, "first-duplicate.plist", "First", "Renamed", "1.0")
            self.write_pkginfo(pkgsinfo, "second.plist", "Second", "Friendly", "1.0")

            entries, skipped = pkginfo_entries(repo)

        self.assertEqual(skipped, 1)
        self.assertEqual({entry[1]["name"] for entry in entries}, {"First", "Second"})

    @staticmethod
    def write_pkginfo(directory, filename, name, display_name, version):
        with open(os.path.join(directory, filename), "wb") as handle:
            plistlib.dump(
                {"name": name, "display_name": display_name, "version": version},
                handle,
            )


class AppUploaderTests(unittest.TestCase):
    def test_display_name_is_optional_and_not_derived_from_name(self):
        uploader = WoodstarMunkiAppUploader()
        uploader.env = {}

        metadata = uploader.software_metadata({"name": "InternalName"}, "InternalName")

        self.assertEqual(metadata["display_name"], "")

    def test_name_does_not_use_generic_recipe_name(self):
        uploader = WoodstarMunkiAppUploader()
        uploader.env = {"NAME": "Recipe Name"}

        with self.assertRaises(ProcessorError):
            uploader.software_name({})

    def test_targets_default_only_when_fields_are_absent(self):
        uploader = WoodstarMunkiAppUploader()
        uploader.env = {"targets": {}}

        self.assertEqual(
            uploader.target_body(None, None),
            {"include": [], "exclude": []},
        )

        for targets in ({"include": {}}, {"exclude": {}}):
            uploader.env = {"targets": targets}
            with self.subTest(targets=targets), self.assertRaises(ProcessorError):
                uploader.target_body(None, None)

    def test_package_selector_rejects_falsey_non_dictionary(self):
        uploader = WoodstarMunkiAppUploader()

        self.assertEqual(
            uploader.package_selector({}),
            {"strategy": "latest"},
        )
        with self.assertRaises(ProcessorError):
            uploader.package_selector({"package": []})

    def test_icon_only_upload_reports_updated_action(self):
        client = FakeWoodstarClient()
        software = client.post(
            "/api/munki/software",
            {
                "name": "com.example.app",
                "display_name": "Example App",
                "description": "",
                "category": "",
                "developer": "",
                "targets": {"include": [], "exclude": []},
            },
        )
        software["icon_object_id"] = 41
        software["icon_file"] = {
            "filename": "old.png",
            "size_bytes": 1,
            "sha256": "old",
        }
        client.software[software["id"]] = software

        with tempfile.NamedTemporaryFile(suffix=".png") as icon:
            icon.write(b"new icon")
            icon.flush()
            uploader = WoodstarMunkiAppUploader()
            uploader.env = {
                "WOODSTAR_URL": "https://woodstar.example",
                "WOODSTAR_API_KEY": "test-key",
                "pkginfo": {
                    "name": "com.example.app",
                    "display_name": "Example App",
                },
                "icon_path": icon.name,
            }
            with patch.object(app_uploader_module, "client_from_env", return_value=client):
                uploader.main()

        summary = uploader.env["woodstarmunkiappuploader_summary_result"]
        self.assertEqual(summary["data"]["action"], "Updated")


class PackageUploaderTests(unittest.TestCase):
    def test_pkginfo_defaults_apply_only_when_fields_are_absent(self):
        self.assertEqual(WoodstarMunkiPackageUploader.installer_type({}), "pkg")
        self.assertEqual(WoodstarMunkiPackageUploader.installer_type({"installer_type": ""}), "")
        self.assertEqual(
            WoodstarMunkiPackageUploader.install_items([{"path": "/tmp/item"}])[0]["type"],
            "file",
        )
        self.assertEqual(
            WoodstarMunkiPackageUploader.install_items(
                [{"type": "", "path": "/tmp/item"}]
            )[0]["type"],
            "",
        )
        with self.assertRaises(ProcessorError):
            WoodstarMunkiPackageUploader.uninstall_method({"uninstall_method": 0})

    def test_installer_choices_accept_only_canonical_munki_keys(self):
        self.assertEqual(
            WoodstarMunkiPackageUploader.installer_choices_xml(
                [
                    {
                        "choiceIdentifier": "com.example.choice",
                        "choiceAttribute": "selected",
                        "attributeSetting": 1,
                    }
                ]
            ),
            [
                {
                    "choice_identifier": "com.example.choice",
                    "choice_attribute": "selected",
                    "attribute_setting": 1,
                }
            ],
        )

        with self.assertRaises(ProcessorError):
            WoodstarMunkiPackageUploader.installer_choices_xml(
                [
                    {
                        "choice_identifier": "com.example.choice",
                        "choice_attribute": "selected",
                        "attribute_setting": 1,
                    }
                ]
            )

    def test_package_installer_upload_precedes_create_and_supplies_object_id(self):
        client = FakeWoodstarClient()
        software = client.post("/api/munki/software", self.software_body())
        client.operations.clear()
        uploader = WoodstarMunkiPackageUploader()
        body = default_package_mutation()
        body.update({"software_id": software["id"], "version": "1.0"})

        with tempfile.NamedTemporaryFile(suffix=".pkg") as artifact:
            artifact.write(b"package-v1")
            artifact.flush()
            package, action, changed, uploaded = uploader.save_package(
                client, software["id"], body, artifact.name
            )

        self.assertEqual([operation[0] for operation in client.operations], ["upload", "post-package"])
        self.assertEqual(package["installer_object_id"], 1)
        self.assertEqual((action, changed, uploaded), ("Created", True, True))

    def test_unchanged_installer_is_reused_before_update_comparison(self):
        client = FakeWoodstarClient()
        software = client.post("/api/munki/software", self.software_body())
        uploader = WoodstarMunkiPackageUploader()
        body = default_package_mutation()
        body.update({"software_id": software["id"], "version": "1.0"})

        with tempfile.NamedTemporaryFile(suffix=".pkg") as artifact:
            artifact.write(b"package-v1")
            artifact.flush()
            uploader.save_package(client, software["id"], body, artifact.name)
            client.operations.clear()
            package, action, changed, uploaded = uploader.save_package(
                client, software["id"], body, artifact.name
            )

        self.assertEqual(client.operations, [])
        self.assertEqual(package["installer_object_id"], 1)
        self.assertEqual((action, changed, uploaded), ("Skipped", False, False))

    def test_failed_package_mutation_deletes_new_installer(self):
        client = FakeWoodstarClient()
        software = client.post("/api/munki/software", self.software_body())
        uploader = WoodstarMunkiPackageUploader()
        body = default_package_mutation()
        body.update({"software_id": software["id"], "version": "1.0"})
        client.fail_package_mutation = True

        with tempfile.NamedTemporaryFile(suffix=".pkg") as artifact:
            artifact.write(b"package-v1")
            artifact.flush()
            with self.assertRaises(ProcessorError):
                uploader.save_package(client, software["id"], body, artifact.name)

        self.assertEqual(
            [operation[0] for operation in client.operations[-3:]],
            ["upload", "post-package", "delete-installer"],
        )

    def test_failed_update_keeps_old_installer_and_deletes_replacement(self):
        client = FakeWoodstarClient()
        software = client.post("/api/munki/software", self.software_body())
        uploader = WoodstarMunkiPackageUploader()
        body = default_package_mutation()
        body.update({"software_id": software["id"], "version": "1.0"})

        with tempfile.NamedTemporaryFile(suffix=".pkg") as artifact:
            artifact.write(b"package-v1")
            artifact.flush()
            original, _action, _changed, _uploaded = uploader.save_package(
                client, software["id"], body, artifact.name
            )
            artifact.seek(0)
            artifact.truncate()
            artifact.write(b"package-v2")
            artifact.flush()
            client.operations.clear()
            client.fail_package_mutation = True
            with self.assertRaises(ProcessorError):
                uploader.save_package(client, software["id"], body, artifact.name)

        self.assertEqual(
            [operation[0] for operation in client.operations],
            ["upload", "put-package", "delete-installer"],
        )
        self.assertEqual(client.packages[original["id"]]["installer_object_id"], 1)

    def test_nopkg_creates_without_installer_upload(self):
        client = FakeWoodstarClient()
        software = client.post("/api/munki/software", self.software_body())
        client.operations.clear()
        uploader = WoodstarMunkiPackageUploader()
        body = default_package_mutation()
        body.update(
            {"software_id": software["id"], "version": "1.0", "installer_type": "nopkg"}
        )

        package, action, changed, uploaded = uploader.save_package(
            client, software["id"], body, None
        )

        self.assertNotIn("installer_object_id", package)
        self.assertEqual([operation[0] for operation in client.operations], ["post-package"])
        self.assertEqual((action, changed, uploaded), ("Created", True, False))

    @staticmethod
    def software_body():
        return {
            "name": "Example",
            "display_name": "",
            "description": "",
            "category": "",
            "developer": "",
            "targets": {"include": [], "exclude": []},
        }


class RepoImporterTests(unittest.TestCase):
    def test_preflight_rejects_divergent_software_metadata(self):
        importer = WoodstarMunkiRepoImporter()
        entries = [
            ("first.plist", {"name": "Example", "display_name": "Example App", "version": "1.0"}),
            ("second.plist", {"name": "Example", "display_name": "Renamed App", "version": "2.0"}),
        ]

        with self.assertRaises(ProcessorError):
            importer.validate_software_metadata(entries)

    def test_preflight_rejects_missing_relation_before_writes(self):
        client = FakeWoodstarClient()
        importer = WoodstarMunkiRepoImporter()
        pkginfo = {
            "name": "Consumer",
            "version": "1.0",
            "installer_type": "nopkg",
            "requires": ["Missing-1.0"],
        }

        with tempfile.TemporaryDirectory() as repo:
            with self.assertRaises(ProcessorError):
                importer.preflight_entries(client, repo, [("consumer.plist", pkginfo)])

        self.assertEqual(client.software, {})
        self.assertEqual(client.packages, {})

    def test_two_phase_import_resolves_later_versioned_dependency_and_is_idempotent(self):
        client = FakeWoodstarClient()
        importer = WoodstarMunkiRepoImporter()
        importer.env = {}
        consumer = {
            "name": "Consumer",
            "display_name": "Consumer App",
            "version": "1.0",
            "installer_type": "nopkg",
            "requires": ["Dependency-2.0"],
        }
        dependency = {
            "name": "Dependency",
            "display_name": "Dependency App",
            "version": "2.0",
            "installer_type": "nopkg",
        }

        with tempfile.TemporaryDirectory() as repo:
            first_counts = self.import_entries(importer, client, repo, [consumer, dependency])
            second_counts = self.import_entries(importer, client, repo, [consumer, dependency])

        consumer_package = next(
            package
            for package in client.packages.values()
            if package["software_name"] == "Consumer"
        )
        dependency_package = next(
            package
            for package in client.packages.values()
            if package["software_name"] == "Dependency"
        )
        consumer_software = client.software[consumer_package["software_id"]]
        dependency_software = client.software[dependency_package["software_id"]]
        self.assertEqual(consumer_software["name"], "Consumer")
        self.assertEqual(consumer_software["display_name"], "Consumer App")
        self.assertEqual(dependency_software["name"], "Dependency")
        self.assertEqual(dependency_software["display_name"], "Dependency App")
        self.assertEqual(
            consumer_package["requires"],
            [
                {
                    "software_id": dependency_package["software_id"],
                    "package_id": dependency_package["id"],
                }
            ],
        )
        self.assertEqual(first_counts["packages_created"], 2)
        self.assertEqual(second_counts["packages_created"], 0)
        self.assertEqual(second_counts["packages_updated"], 0)

    @staticmethod
    def import_entries(importer, client, repo, pkginfos):
        counts = import_counts()
        entries = [
            (f"item-{index}.plist", pkginfo)
            for index, pkginfo in enumerate(pkginfos)
        ]
        importer.preflight_entries(client, repo, entries)
        prepared = [
            importer.prepare_pkginfo(client, repo, path, pkginfo, counts)
            for path, pkginfo in entries
        ]
        resolver = PackageReferenceResolver.from_client(client)
        for item in prepared:
            importer.finish_pkginfo(client, item, resolver, counts)
        return counts


class FakeWoodstarClient:
    def __init__(self):
        self.software = {}
        self.packages = {}
        self.next_software_id = 1
        self.next_package_id = 1
        self.next_object_id = 1
        self.objects = {}
        self.operations = []
        self.fail_package_mutation = False

    def get(self, path, query=None):
        if path == "/api/munki/software":
            return self.page(self.filter_items(self.software.values(), query))
        if path == "/api/munki/packages":
            items = self.filter_items(self.packages.values(), query)
            if query and query.get("software_id"):
                items = [item for item in items if item["software_id"] == query["software_id"]]
            return self.page(items)
        if path.startswith("/api/munki/software/"):
            return dict(self.software[int(path.rsplit("/", 1)[1])])
        if path.startswith("/api/munki/packages/"):
            return dict(self.packages[int(path.rsplit("/", 1)[1])])
        raise AssertionError(f"unexpected GET {path}")

    def post(self, path, body=None):
        if path == "/api/munki/software":
            item = dict(body)
            item["id"] = self.next_software_id
            item.setdefault("targets", {"include": [], "exclude": []})
            self.software[item["id"]] = item
            self.next_software_id += 1
            return dict(item)
        if path == "/api/munki/packages":
            self.operations.append(("post-package", dict(body)))
            if self.fail_package_mutation:
                raise ProcessorError("package mutation failed")
            item = dict(body)
            item["id"] = self.next_package_id
            item["software_name"] = self.software[item["software_id"]]["name"]
            item.setdefault("requires", [])
            item.setdefault("update_for", [])
            self.add_installer_metadata(item)
            self.packages[item["id"]] = item
            self.next_package_id += 1
            return dict(item)
        raise AssertionError(f"unexpected POST {path}")

    def put(self, path, body=None):
        if path.startswith("/api/munki/packages/"):
            package_id = int(path.rsplit("/", 1)[1])
            self.operations.append(("put-package", package_id, dict(body)))
            if self.fail_package_mutation:
                raise ProcessorError("package mutation failed")
            software_id = self.packages[package_id]["software_id"]
            software_name = self.packages[package_id]["software_name"]
            self.packages[package_id] = {
                **dict(body),
                "id": package_id,
                "software_id": software_id,
                "software_name": software_name,
            }
            self.add_installer_metadata(self.packages[package_id])
            return dict(self.packages[package_id])
        raise AssertionError(f"unexpected PUT {path}")

    def attach_object(self, path, file_path):
        if "/api/munki/software/" not in path or not path.endswith("/icon"):
            raise AssertionError(f"unexpected attachment {path}")
        software_id = int(path.split("/")[4])
        self.software[software_id]["icon_object_id"] = 42
        self.software[software_id]["icon_file"] = local_file_metadata(file_path)

    def upload_package_installer(self, file_path):
        object_id = self.next_object_id
        self.next_object_id += 1
        metadata = local_file_metadata(file_path)
        self.objects[object_id] = metadata
        self.operations.append(("upload", object_id))
        return {"id": object_id, **metadata}

    def delete(self, path):
        prefix = "/api/munki/package-installers/"
        if not path.startswith(prefix):
            raise AssertionError(f"unexpected DELETE {path}")
        object_id = int(path.removeprefix(prefix))
        self.operations.append(("delete-installer", object_id))
        self.objects.pop(object_id, None)

    def add_installer_metadata(self, package):
        object_id = package.get("installer_object_id")
        if object_id is None:
            package.pop("installer_file", None)
            return
        package["installer_file"] = dict(self.objects[object_id])

    @staticmethod
    def page(items):
        return {"items": [dict(item) for item in items], "count": len(items)}

    @staticmethod
    def filter_items(items, query):
        items = list(items)
        if not query or not query.get("q"):
            return items
        value = str(query["q"])
        return [
            item
            for item in items
            if value in str(item.get("name", ""))
            or value in str(item.get("version", ""))
        ]


if __name__ == "__main__":
    unittest.main()
