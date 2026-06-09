#!/usr/local/autopkg/python

import os.path
import plistlib
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, truthy  # noqa: E402

__all__ = ["WoodstarMunkiPackageUploader"]

PACKAGE_DIRECT_KEYS = (
    "unattended_install",
    "unattended_uninstall",
    "minimum_munki_version",
    "minimum_os_version",
    "maximum_os_version",
    "supported_architectures",
    "blocking_applications",
    "installable_condition",
    "blocking_applications_manual_quit_only",
    "blocking_applications_quit_script",
    "precache",
    "autoremove",
    "apple_item",
    "suppress_bundle_relocation",
    "force_install_after_date",
    "installed_size",
    "package_path",
    "installer_choices_xml",
    "items_to_copy",
    "notes",
    "installcheck_script",
    "uninstallcheck_script",
    "preinstall_script",
    "postinstall_script",
    "preuninstall_script",
    "postuninstall_script",
    "uninstall_script",
    "version_script",
)


class WoodstarMunkiPackageUploader(Processor):
    description = "Uploads Munki package artifacts and creates or updates a Woodstar package."

    input_variables = {
        "WOODSTAR_URL": {
            "required": True,
            "description": "Woodstar base URL, for example http://localhost:8080.",
        },
        "WOODSTAR_API_KEY": {
            "required": True,
            "description": "Woodstar admin API key.",
        },
        "MUNKI_REPO": {
            "required": True,
            "description": "Munki repo path containing MunkiImporter output.",
        },
        "pkginfo_repo_path": {
            "required": True,
            "description": "MunkiImporter pkginfo output path.",
        },
        "pkg_repo_path": {
            "required": False,
            "description": "MunkiImporter package artifact path. Required for package-bearing pkginfo.",
        },
        "uninstaller_pkg_path": {
            "required": False,
            "description": "Optional uninstaller package path for uninstall_package pkginfo.",
        },
        "software_id": {
            "required": False,
            "description": "Existing Woodstar Munki software ID. Defaults to woodstar_software_id.",
        },
        "eligible": {
            "required": False,
            "description": "Whether this package is eligible when targets request the latest package.",
            "default": True,
        },
    }
    output_variables = {
        "woodstar_package_artifact": {"description": "Uploaded package artifact response."},
        "woodstar_uninstaller_package_artifact": {
            "description": "Uploaded uninstaller package artifact response.",
        },
        "woodstar_package": {"description": "Woodstar package response."},
        "woodstar_package_id": {"description": "Woodstar package ID."},
        "woodstarmunkipackageuploader_summary_result": {
            "description": "Summary of Woodstar package upload.",
        },
    }

    def main(self):
        client = client_from_env(self.env)
        pkginfo = self.pkginfo()
        software_id = self.software_id()
        installer_path, uninstaller_path = self.artifact_paths(pkginfo)

        installer_artifact = None
        if installer_path:
            installer_artifact = client.upload_artifact("package", installer_path, os.path.basename(installer_path))

        uninstaller_artifact = None
        if uninstaller_path:
            uninstaller_artifact = client.upload_artifact(
                "package",
                uninstaller_path,
                os.path.basename(uninstaller_path),
            )

        body = self.package_body(pkginfo, software_id, installer_artifact, uninstaller_artifact)
        package, action = self.save_package(client, software_id, body)
        if installer_artifact:
            self.env["woodstar_package_artifact"] = installer_artifact
        if uninstaller_artifact:
            self.env["woodstar_uninstaller_package_artifact"] = uninstaller_artifact
        self.env["woodstar_package"] = package
        self.env["woodstar_package_id"] = package["id"]
        self.env["woodstarmunkipackageuploader_summary_result"] = {
            "summary_text": "Woodstar Munki package uploaded",
            "report_fields": ["id", "software", "version", "artifact", "uninstaller_artifact"],
            "data": {
                "id": str(package["id"]),
                "software": package.get("software_name", ""),
                "version": package["version"],
                "artifact": installer_artifact["location"] if installer_artifact else "",
                "uninstaller_artifact": uninstaller_artifact["location"] if uninstaller_artifact else "",
            },
        }
        self.output(
            f"{action} Woodstar package {package['id']}: {package.get('software_name', '')} {package['version']}")

    def software_id(self):
        software_id = self.env.get("software_id") or self.env.get("woodstar_software_id")
        if not software_id:
            raise ProcessorError("software_id or woodstar_software_id is required")
        try:
            return int(software_id)
        except (TypeError, ValueError) as err:
            raise ProcessorError("software_id or woodstar_software_id must be an integer") from err

    def load_pkginfo(self, path):
        with open(path, "rb") as handle:
            return plistlib.load(handle)

    def repo_pkginfo_path(self):
        path = self.repo_path(self.env.get("pkginfo_repo_path"))
        if not path:
            raise ProcessorError("pkginfo_repo_path is required from MunkiImporter")
        if not os.path.isfile(path):
            raise ProcessorError(f"MunkiImporter pkginfo not found: {path}")
        return path

    def pkginfo(self):
        pkginfo_path = self.repo_pkginfo_path()
        self.output(f"Using MunkiImporter pkginfo: {pkginfo_path}")
        try:
            return self.load_pkginfo(pkginfo_path)
        except (OSError, plistlib.InvalidFileException, ValueError) as err:
            raise ProcessorError(f"failed to read MunkiImporter pkginfo {pkginfo_path}: {err}") from err

    def package_path(self):
        path = self.repo_path(self.env.get("pkg_repo_path"))
        if path and os.path.isfile(path):
            return path
        raise ProcessorError("pkg_repo_path is required for package-bearing MunkiImporter pkginfo")

    def uninstaller_package_path(self):
        path = self.repo_path(self.env.get("uninstaller_pkg_path"))
        if path and os.path.isfile(path):
            return path
        raise ProcessorError("uninstaller_pkg_path is required for uninstall_package pkginfo")

    def artifact_paths(self, pkginfo):
        installer_path = None
        if self.needs_installer_artifact(pkginfo):
            installer_path = self.package_path()

        uninstaller_path = None
        if self.needs_uninstaller_artifact(pkginfo):
            uninstaller_path = self.uninstaller_package_path()
        return installer_path, uninstaller_path

    def repo_path(self, value):
        if not value:
            return None
        value = str(value)
        if os.path.isabs(value):
            return value
        munki_repo = self.env.get("MUNKI_REPO")
        if munki_repo:
            return os.path.join(munki_repo, value)
        return value

    def needs_installer_artifact(self, pkginfo):
        return self.installer_type(pkginfo) != "nopkg"

    def needs_uninstaller_artifact(self, pkginfo):
        return self.uninstall_method(pkginfo) == "uninstall_package"

    def package_body(self, pkginfo, software_id, installer_artifact, uninstaller_artifact):
        body = self.package_mutation_from_pkginfo(pkginfo)
        body["software_id"] = software_id
        body["eligible"] = truthy(self.env.get("eligible", True))
        if installer_artifact:
            body["installer_artifact_id"] = installer_artifact["id"]
        if uninstaller_artifact:
            body["uninstaller_artifact_id"] = uninstaller_artifact["id"]
        return body

    def package_mutation_from_pkginfo(self, pkginfo):
        if not isinstance(pkginfo, dict):
            raise ProcessorError("pkginfo must be a dictionary")
        version = pkginfo.get("version")
        if not isinstance(version, str) or version == "":
            raise ProcessorError("pkginfo version is required")

        body = {
            "version": version,
            "installer_type": self.installer_type(pkginfo),
        }
        uninstall_method = self.uninstall_method(pkginfo)
        if uninstall_method:
            body["uninstall_method"] = uninstall_method
        for key in PACKAGE_DIRECT_KEYS:
            if key in pkginfo:
                body[key] = pkginfo[key]
        if "RestartAction" in pkginfo:
            body["restart_action"] = pkginfo["RestartAction"]
        if "OnDemand" in pkginfo:
            body["on_demand"] = pkginfo["OnDemand"]
        if "installer_environment" in pkginfo:
            body["installer_environment"] = self.installer_environment(pkginfo["installer_environment"])
        if "installs" in pkginfo:
            body["installs"] = self.install_items(pkginfo["installs"])
        if "receipts" in pkginfo:
            body["receipts"] = self.receipts(pkginfo["receipts"])
        if "requires" in pkginfo:
            body["requires"] = self.package_references(pkginfo["requires"], "requires")
        if "update_for" in pkginfo:
            body["update_for"] = self.package_references(pkginfo["update_for"], "update_for")
        for munki_key, woodstar_key in (
            ("preinstall_alert", "preinstall_alert"),
            ("preuninstall_alert", "preuninstall_alert"),
        ):
            if munki_key in pkginfo:
                body[woodstar_key] = self.alert(pkginfo[munki_key], munki_key)
        return body

    def save_package(self, client, software_id, body):
        existing = self.existing_package(client, software_id, body["version"])
        if existing:
            update_body = dict(body)
            del update_body["software_id"]
            return client.put(f"/api/munki/packages/{existing['id']}", update_body), "Updated"
        return client.post("/api/munki/packages", body), "Created"

    @staticmethod
    def existing_package(client, software_id, version):
        page = client.get(
            "/api/munki/packages",
            {"software_id": software_id, "q": version, "page_size": 1000},
        )
        for item in page.get("items") or []:
            if item.get("software_id") == software_id and item.get("version") == version:
                return item
        return None

    @staticmethod
    def installer_type(pkginfo):
        value = pkginfo.get("installer_type") or "pkg"
        if not isinstance(value, str):
            raise ProcessorError("pkginfo installer_type must be a string")
        return value

    @staticmethod
    def uninstall_method(pkginfo):
        value = pkginfo.get("uninstall_method") or ""
        if not isinstance(value, str):
            raise ProcessorError("pkginfo uninstall_method must be a string")
        return value

    @staticmethod
    def installer_environment(value):
        if not isinstance(value, dict):
            raise ProcessorError("pkginfo installer_environment must be a dictionary")
        return [{"name": name, "value": value[name]} for name in sorted(value)]

    @staticmethod
    def install_items(values):
        if not isinstance(values, list):
            raise ProcessorError("pkginfo installs must be a list")
        items = []
        for value in values:
            if not isinstance(value, dict):
                raise ProcessorError("pkginfo installs entries must be dictionaries")
            item = {
                "type": value.get("type") or "file",
                "path": value.get("path"),
            }
            for munki_key, woodstar_key in (
                ("CFBundleIdentifier", "bundle_identifier"),
                ("CFBundleName", "bundle_name"),
                ("CFBundleShortVersionString", "bundle_short_version"),
                ("CFBundleVersion", "bundle_version"),
                ("version_comparison_key", "version_comparison_key"),
                ("md5checksum", "md5checksum"),
                ("minimum_os_version", "minimum_os_version"),
                ("installer_item_location", "installer_item_location"),
            ):
                if munki_key in value:
                    item[woodstar_key] = value[munki_key]
            items.append(item)
        return items

    @staticmethod
    def receipts(values):
        if not isinstance(values, list):
            raise ProcessorError("pkginfo receipts must be a list")
        receipts = []
        for value in values:
            if not isinstance(value, dict):
                raise ProcessorError("pkginfo receipts entries must be dictionaries")
            receipt = {"package_id": value.get("packageid")}
            for key in ("version", "optional"):
                if key in value:
                    receipt[key] = value[key]
            receipts.append(receipt)
        return receipts

    @staticmethod
    def package_references(values, key):
        if not isinstance(values, list):
            raise ProcessorError(f"pkginfo {key} must be a list")
        references = []
        for value in values:
            try:
                package_id = int(value)
            except (TypeError, ValueError) as err:
                raise ProcessorError(f"pkginfo {key} entries must be Woodstar package IDs") from err
            if package_id <= 0:
                raise ProcessorError(f"pkginfo {key} entries must be Woodstar package IDs")
            references.append({"package_id": package_id})
        return references

    @staticmethod
    def alert(value, key):
        if not isinstance(value, dict):
            raise ProcessorError(f"pkginfo {key} must be a dictionary")
        alert = {"enabled": False}
        for munki_key, woodstar_key in (
            ("alert_title", "title"),
            ("alert_detail", "detail"),
            ("ok_label", "ok_label"),
            ("cancel_label", "cancel_label"),
        ):
            if munki_key in value:
                alert[woodstar_key] = value[munki_key]
                alert["enabled"] = True
        return alert


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiPackageUploader()
    PROCESSOR.execute_shell()
