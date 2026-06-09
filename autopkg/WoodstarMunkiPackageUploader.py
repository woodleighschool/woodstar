#!/usr/local/autopkg/python

import os.path
import plistlib
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, truthy  # noqa: E402

__all__ = ["WoodstarMunkiPackageUploader"]

SUPPORTED_INSTALLER_TYPES = {"", "pkg", "nopkg", "copy_from_dmg"}
SUPPORTED_UNINSTALL_METHODS = {
    "",
    "none",
    "removepackages",
    "remove_copied_items",
    "uninstall_script",
    "uninstall_package",
}
class WoodstarMunkiPackageUploader(Processor):
    description = "Uploads a Munki package artifact and imports pkginfo into Woodstar."

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
        "MUNKI_REPO_SUBDIR": {
            "required": False,
            "description": "MunkiImporter repo subdirectory containing the generated pkginfo.",
        },
        "version": {
            "required": False,
            "description": "Package version used by MunkiImporter when naming the generated pkginfo.",
        },
        "pkg_repo_path": {
            "required": False,
            "description": "MunkiImporter package artifact path. Required for package-bearing pkginfo.",
        },
        "uninstaller_pkg_path": {
            "required": False,
            "description": "Optional uninstaller package path for uninstall_package pkginfo.",
        },
        "pkginfo": {
            "required": True,
            "description": "Recipe pkginfo used to locate the MunkiImporter repo pkginfo by name.",
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
        "woodstar_package": {"description": "Imported Woodstar package response."},
        "woodstar_package_id": {"description": "Imported Woodstar package ID."},
        "woodstarmunkipackageuploader_summary_result": {
            "description": "Summary of Woodstar package upload.",
        },
    }

    def main(self):
        client = client_from_env(self.env)
        pkginfo = self.pkginfo()
        software_id = self.software_id()
        self.validate_pkginfo(pkginfo)
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

        body = {
            "pkginfo": pkginfo,
            "eligible": truthy(self.env.get("eligible", True)),
        }
        if installer_artifact:
            body["installer_artifact_id"] = installer_artifact["id"]
        if uninstaller_artifact:
            body["uninstaller_artifact_id"] = uninstaller_artifact["id"]

        package = client.post(f"/api/munki/software/{software_id}/packages/import", body)
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
            "Imported Woodstar package "
            f"{package['id']}: {package.get('software_name', '')} {package['version']}"
        )

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
        input_pkginfo = self.env.get("pkginfo")
        if not isinstance(input_pkginfo, dict) or not input_pkginfo.get("name"):
            raise ProcessorError("pkginfo.name is required to locate MunkiImporter output")
        expected_name = input_pkginfo["name"]
        expected_version = self.env.get("version") or ""
        if not expected_version:
            raise ProcessorError("version is required to locate MunkiImporter output")

        subdir = self.env.get("MUNKI_REPO_SUBDIR") or ""
        filename = f"{expected_name}-{expected_version}.plist"
        path = self.repo_path(os.path.join("pkgsinfo", subdir, filename))
        if not path:
            raise ProcessorError("MUNKI_REPO is required to locate MunkiImporter output")
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
        value = self.repo_path(self.env.get("pkg_repo_path"))
        if value and os.path.exists(value):
            return value
        raise ProcessorError("pkg_repo_path is required for package-bearing MunkiImporter pkginfo")

    def uninstaller_package_path(self):
        for key in ("uninstaller_pkg_path", "uninstaller_pkg_repo_path"):
            value = self.repo_path(self.env.get(key))
            if value and os.path.exists(value):
                return value
        raise ProcessorError("uninstaller_pkg_path is required for uninstall_package pkginfo")

    def artifact_paths(self, pkginfo):
        installer_path = None
        if self.needs_installer_artifact(pkginfo):
            installer_path = self.package_path()
        elif self.package_path_is_set():
            raise ProcessorError("nopkg installer_type cannot upload pkg_repo_path")

        uninstaller_path = None
        if self.needs_uninstaller_artifact(pkginfo):
            uninstaller_path = self.uninstaller_package_path()
        return installer_path, uninstaller_path

    def package_path_is_set(self):
        return bool(self.env.get("pkg_repo_path"))

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

    def validate_pkginfo(self, pkginfo):
        if not str(pkginfo.get("version") or "").strip():
            raise ProcessorError("pkginfo version is required")
        installer_type = self.installer_type(pkginfo)
        if installer_type not in SUPPORTED_INSTALLER_TYPES:
            raise ProcessorError(f"installer_type {installer_type!r} is not supported by Woodstar")
        if installer_type == "copy_from_dmg" and not self.non_empty_list(pkginfo, "items_to_copy"):
            raise ProcessorError("copy_from_dmg installer_type requires items_to_copy")
        if installer_type == "nopkg" and not self.has_nopkg_evidence(pkginfo):
            raise ProcessorError("nopkg installer_type requires installcheck_script, installs, receipts, or OnDemand")
        uninstall_method = str(pkginfo.get("uninstall_method") or "").strip()
        if uninstall_method not in SUPPORTED_UNINSTALL_METHODS:
            raise ProcessorError(f"uninstall_method {uninstall_method!r} is not supported by Woodstar")
        if uninstall_method == "removepackages" and not self.non_empty_list(pkginfo, "receipts"):
            raise ProcessorError("removepackages uninstall_method requires receipts")
        if uninstall_method == "remove_copied_items" and not self.non_empty_list(pkginfo, "items_to_copy"):
            raise ProcessorError("remove_copied_items uninstall_method requires items_to_copy")
        if uninstall_method == "uninstall_script" and not str(pkginfo.get("uninstall_script") or "").strip():
            raise ProcessorError("uninstall_script uninstall_method requires uninstall_script")
        for key in ("requires", "update_for"):
            self.validate_package_references(pkginfo, key)

    def needs_installer_artifact(self, pkginfo):
        return self.installer_type(pkginfo) != "nopkg"

    @staticmethod
    def needs_uninstaller_artifact(pkginfo):
        return str(pkginfo.get("uninstall_method") or "").strip() == "uninstall_package"

    @staticmethod
    def installer_type(pkginfo):
        return str(pkginfo.get("installer_type") or "pkg").strip()

    @staticmethod
    def non_empty_list(pkginfo, key):
        value = pkginfo.get(key)
        return isinstance(value, list) and len(value) > 0

    @staticmethod
    def has_nopkg_evidence(pkginfo):
        return (
            bool(str(pkginfo.get("installcheck_script") or "").strip())
            or WoodstarMunkiPackageUploader.non_empty_list(pkginfo, "installs")
            or WoodstarMunkiPackageUploader.non_empty_list(pkginfo, "receipts")
            or truthy(pkginfo.get("OnDemand", pkginfo.get("on_demand", False)))
        )

    @staticmethod
    def validate_package_references(pkginfo, key):
        references = pkginfo.get(key) or []
        if not isinstance(references, list):
            raise ProcessorError(f"{key} must be a list of Woodstar package IDs")
        for reference in references:
            try:
                package_id = int(str(reference).strip())
            except (TypeError, ValueError) as err:
                raise ProcessorError(f"{key} entries must be Woodstar package IDs") from err
            if package_id <= 0:
                raise ProcessorError(f"{key} entries must be Woodstar package IDs")


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiPackageUploader()
    PROCESSOR.execute_shell()
