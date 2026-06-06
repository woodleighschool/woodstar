#!/usr/local/autopkg/python

import json
import os.path
import plistlib
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, truthy  # noqa: E402

__all__ = ["WoodstarMunkiPackageUploader"]


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
            "required": False,
            "description": "Munki repo path used to resolve pkg_repo_path/pkginfo_repo_path.",
        },
        "pkg_path": {
            "required": False,
            "description": "Package path. Defaults to MunkiImporter pkg_repo_path.",
        },
        "pkginfo": {
            "required": False,
            "description": "Munki pkginfo dictionary. Defaults to munki_info.",
        },
        "software_id": {
            "required": False,
            "description": "Existing Woodstar software title ID. Defaults to woodstar_software_id.",
        },
        "eligible": {
            "required": False,
            "description": "Whether this package is eligible for latest_eligible assignments.",
            "default": True,
        },
    }
    output_variables = {
        "woodstar_package_artifact": {"description": "Uploaded package artifact response."},
        "woodstar_package": {"description": "Imported Woodstar package response."},
        "woodstar_package_id": {"description": "Imported Woodstar package ID."},
        "woodstarmunkipackageuploader_summary_result": {
            "description": "Summary of Woodstar package upload.",
        },
    }

    def main(self):
        client = client_from_env(self.env)
        pkginfo = self.pkginfo()
        pkg_path = self.package_path()
        artifact = client.upload_artifact("package", pkg_path, os.path.basename(pkg_path))

        body = {
            "pkginfo": pkginfo,
            "installer_artifact_id": artifact["id"],
            "eligible": truthy(self.env.get("eligible", True)),
        }
        software_id = self.env.get("software_id") or self.env.get("woodstar_software_id")
        if software_id:
            body["software_id"] = int(software_id)

        package = client.post("/api/munki/packages/import", body)
        self.env["woodstar_package_artifact"] = artifact
        self.env["woodstar_package"] = package
        self.env["woodstar_package_id"] = package["id"]
        self.env["woodstarmunkipackageuploader_summary_result"] = {
            "summary_text": "Woodstar Munki package uploaded",
            "report_fields": ["id", "name", "version", "artifact"],
            "data": {
                "id": str(package["id"]),
                "name": package["name"],
                "version": package["version"],
                "artifact": artifact["location"],
            },
        }
        self.output(
            f"Uploaded Woodstar package {package['id']}: {package['name']} {package['version']}"
        )

    def pkginfo(self):
        pkginfo = self.env.get("munki_info") or self.env.get("pkginfo")
        if isinstance(pkginfo, dict):
            return self.with_version(pkginfo)
        pkginfo_path = self.repo_path(self.env.get("pkginfo_repo_path"))
        if pkginfo_path:
            with open(pkginfo_path, "rb") as handle:
                return self.with_version(plistlib.load(handle))
        raise ProcessorError("pkginfo, munki_info, or pkginfo_repo_path is required")

    def package_path(self):
        for key in ("pathname", "pkg_path", "pkg_repo_path"):
            value = self.repo_path(self.env.get(key))
            if value and os.path.exists(value):
                return value
        raise ProcessorError("pkg_path, pkg_repo_path, or pathname is required")

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

    def with_version(self, pkginfo):
        pkginfo = json.loads(json.dumps(pkginfo, default=str))
        if not pkginfo.get("version") and self.env.get("version"):
            pkginfo["version"] = self.env["version"]
        pkginfo.pop("icon_name", None)
        pkginfo.pop("icon_hash", None)
        return pkginfo


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiPackageUploader()
    PROCESSOR.execute_shell()
