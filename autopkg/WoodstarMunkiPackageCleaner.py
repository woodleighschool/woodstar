#!/usr/local/autopkg/python

import os.path
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, list_items  # noqa: E402

__all__ = ["WoodstarMunkiPackageCleaner"]


class WoodstarMunkiPackageCleaner(Processor):
    description = "Deletes older Woodstar Munki packages for one software item."

    input_variables = {
        "WOODSTAR_URL": {
            "required": True,
            "description": "Woodstar base URL, for example http://localhost:8080.",
        },
        "WOODSTAR_API_KEY": {
            "required": True,
            "description": "Woodstar admin API key.",
        },
        "software_id": {
            "required": False,
            "description": "Woodstar Munki software ID. Defaults to woodstar_software_id.",
        },
        "keep_version_count": {
            "required": False,
            "description": "Number of newest package rows to keep.",
            "default": 5,
        },
    }
    output_variables = {
        "woodstar_deleted_package_count": {"description": "Number of package rows deleted."},
        "woodstarmunkipackagecleaner_summary_result": {
            "description": "Summary of deleted Woodstar packages.",
        },
    }

    def main(self):
        self.env.pop("woodstarmunkipackagecleaner_summary_result", None)
        client = client_from_env(self.env)
        software_id = self.software_id()
        keep_count = self.keep_version_count()

        packages = list_items(
            client,
            "/api/munki/packages",
            {"software_id": software_id, "per_page": 1000},
        )
        packages = sorted(packages, key=lambda item: int(item["id"]), reverse=True)
        delete_packages = packages[keep_count:]
        artifact_ids = artifact_ids_for_packages(delete_packages)

        for package in delete_packages:
            client.delete(f"/api/munki/packages/{package['id']}")

        deleted_artifacts = 0
        for artifact_id in artifact_ids:
            if self.delete_artifact(client, artifact_id):
                deleted_artifacts += 1

        deleted_packages = len(delete_packages)
        self.env["woodstar_deleted_package_count"] = deleted_packages
        if deleted_packages:
            self.env["woodstarmunkipackagecleaner_summary_result"] = {
                "summary_text": "Woodstar Munki packages cleaned",
                "report_fields": ["software_id", "deleted_packages", "deleted_artifacts"],
                "data": {
                    "software_id": str(software_id),
                    "deleted_packages": str(deleted_packages),
                    "deleted_artifacts": str(deleted_artifacts),
                },
            }
        self.output(f"Deleted {deleted_packages} Woodstar package(s) for software {software_id}")

    def software_id(self):
        software_id = self.env.get("software_id") or self.env.get("woodstar_software_id")
        if not software_id:
            raise ProcessorError("software_id or woodstar_software_id is required")
        try:
            return int(software_id)
        except (TypeError, ValueError) as err:
            raise ProcessorError("software_id or woodstar_software_id must be an integer") from err

    def keep_version_count(self):
        value = self.env.get("keep_version_count", self.env.get("KEEP_VERSION_COUNT", 5))
        try:
            keep_count = int(value)
        except (TypeError, ValueError) as err:
            raise ProcessorError("keep_version_count must be an integer") from err
        if keep_count < 1:
            raise ProcessorError("keep_version_count must be at least 1")
        return keep_count

    @staticmethod
    def delete_artifact(client, artifact_id):
        try:
            client.delete(f"/api/munki/artifacts/{artifact_id}")
        except ProcessorError as err:
            message = str(err)
            if "HTTP 404" in message or "HTTP 409" in message:
                return False
            raise
        return True


def artifact_ids_for_packages(packages):
    ids = []
    for package in packages:
        for key in ("installer_artifact_id", "uninstaller_artifact_id"):
            artifact_id = package.get(key)
            if artifact_id and artifact_id not in ids:
                ids.append(artifact_id)
    return ids


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiPackageCleaner()
    PROCESSOR.execute_shell()
