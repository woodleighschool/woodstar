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
            "description": "Woodstar HTTPS origin, for example https://woodstar.example.",
        },
        "WOODSTAR_API_KEY": {
            "required": True,
            "description": "Woodstar admin API key.",
        },
        "WOODSTAR_CA_FILE": {
            "required": False,
            "description": "PEM CA file for a private Woodstar certificate chain.",
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

        delete_ids = [str(package["id"]) for package in delete_packages]
        if delete_ids:
            client.delete(
                "/api/munki/packages",
                query={"ids": ",".join(delete_ids)},
            )

        deleted_packages = len(delete_ids)
        self.env["woodstar_deleted_package_count"] = deleted_packages
        if deleted_packages:
            self.env["woodstarmunkipackagecleaner_summary_result"] = {
                "summary_text": "Woodstar Munki packages cleaned",
                "report_fields": ["software_id", "deleted_packages"],
                "data": {
                    "software_id": str(software_id),
                    "deleted_packages": str(deleted_packages),
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


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiPackageCleaner()
    PROCESSOR.execute_shell()
