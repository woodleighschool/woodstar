#!/usr/local/autopkg/python

import os.path
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, find_exact, truthy  # noqa: E402

__all__ = ["WoodstarMunkiAppUploader"]


class WoodstarMunkiAppUploader(Processor):
    description = "Upserts a Woodstar Munki software title, icon, and targets."

    input_variables = {
        "WOODSTAR_URL": {
            "required": True,
            "description": "Woodstar base URL, for example http://localhost:8080.",
        },
        "WOODSTAR_API_KEY": {
            "required": True,
            "description": "Woodstar admin API key.",
        },
        "name": {
            "required": False,
            "description": "Woodstar software title name. Defaults to pkginfo display_name, pkginfo name, or NAME.",
        },
        "description": {"required": False, "description": "Software description."},
        "category": {"required": False, "description": "Software category."},
        "developer": {"required": False, "description": "Software developer."},
        "icon_path": {
            "required": False,
            "description": "Icon file to upload and attach to the software title.",
        },
        "assignments": {
            "required": False,
            "description": "Optional full target object with includes and exclude labels.",
        },
    }
    output_variables = {
        "woodstar_software": {"description": "Software title response."},
        "woodstar_software_id": {"description": "Woodstar software title ID."},
        "woodstar_icon_artifact": {"description": "Uploaded icon artifact response."},
        "woodstar_assignments": {"description": "Software title includes and exclude labels."},
        "woodstarmunkiappuploader_summary_result": {
            "description": "Summary of Woodstar app changes.",
        },
    }

    def main(self):
        client = client_from_env(self.env)
        pkginfo = self.env.get("munki_info") or self.env.get("pkginfo") or {}
        name = self.software_name(pkginfo)
        if not name:
            raise ProcessorError("name is required")

        icon_artifact = None
        icon_path = self.env.get("icon_path")
        if icon_path:
            icon_artifact = client.upload_artifact("icon", icon_path, os.path.basename(icon_path))
            self.env["woodstar_icon_artifact"] = icon_artifact

        software = self.upsert_software(client, pkginfo, name, icon_artifact)
        assignments = {
            "includes": software.get("includes") or [],
            "exclude_label_ids": software.get("exclude_label_ids") or [],
        }

        self.env["woodstar_software"] = software
        self.env["woodstar_software_id"] = software["id"]
        self.env["woodstar_assignments"] = assignments
        self.env["woodstarmunkiappuploader_summary_result"] = {
            "summary_text": "Woodstar Munki app processed",
            "report_fields": ["id", "name", "includes", "exclude_labels"],
            "data": {
                "id": str(software["id"]),
                "name": software["name"],
                "includes": str(len(assignments["includes"])),
                "exclude_labels": str(len(assignments.get("exclude_label_ids") or [])),
            },
        }
        self.output(f"Processed Woodstar software title {software['id']}: {software['name']}")

    def upsert_software(self, client, pkginfo, name, icon_artifact):
        body = {
            "name": name,
            "description": self.env.get("description") or pkginfo.get("description") or "",
            "category": self.env.get("category") or pkginfo.get("category") or "",
            "developer": self.env.get("developer") or pkginfo.get("developer") or "",
        }
        if icon_artifact:
            body["icon_artifact_id"] = icon_artifact["id"]
        existing = find_exact(client, "/api/munki/software-titles", "name", name)
        existing_detail = None
        if existing:
            existing_detail = client.get(f"/api/munki/software-titles/{existing['id']}")
        body.update(self.target_body(client, existing_detail))
        if existing:
            self.output(f"Updating Woodstar software title: {name}")
            return client.patch(f"/api/munki/software-titles/{existing['id']}", body)
        self.output(f"Creating Woodstar software title: {name}")
        return client.post("/api/munki/software-titles", body)

    def software_name(self, pkginfo):
        return (
            self.env.get("name")
            or pkginfo.get("display_name")
            or pkginfo.get("name")
            or self.env.get("NAME")
        )

    def target_body(self, client, existing):
        assignment_config = self.env.get("assignments")
        if not assignment_config:
            return {
                "includes": self.existing_includes(existing),
                "exclude_label_ids": (existing or {}).get("exclude_label_ids") or [],
            }
        if not isinstance(assignment_config, dict):
            raise ProcessorError("assignments must be a dictionary")
        include_inputs = assignment_config.get("includes") or []
        if not isinstance(include_inputs, list):
            raise ProcessorError("assignments.includes must be a list")

        includes = [self.include_body(client, assignment) for assignment in include_inputs]
        exclude_label_ids = self.exclude_label_ids(client, assignment_config)
        return {"includes": includes, "exclude_label_ids": exclude_label_ids}

    def include_body(self, client, assignment):
        if not isinstance(assignment, dict):
            raise ProcessorError("assignments.includes entries must be dictionaries")
        if "priority" in assignment:
            raise ProcessorError("assignments.includes priority is not supported; order the includes list instead")
        label_id = int(self.label_id(client, assignment))
        body = {
            "label_id": label_id,
            "action": assignment.get("action", "install"),
            "optional_install": truthy(assignment.get("optional_install", False)),
            "featured_item": truthy(assignment.get("featured_item", False)),
            "package_selection": assignment.get("package_selection", "latest_eligible"),
        }
        if assignment.get("pinned_package_id"):
            body["pinned_package_id"] = int(assignment["pinned_package_id"])
        return body

    def exclude_label_ids(self, client, assignment_config):
        values = []
        for label_id in assignment_config.get("exclude_label_ids") or []:
            values.append(int(label_id))
        for label_name in assignment_config.get("exclude_label_names") or []:
            values.append(int(self.label_id_by_name(client, label_name)))
        if len(values) != len(set(values)):
            raise ProcessorError("assignments exclude labels contain duplicates")
        return values

    @staticmethod
    def existing_includes(existing):
        includes = []
        for item in (existing or {}).get("includes") or []:
            body = {
                "label_id": item["label_id"],
                "action": item["action"],
                "optional_install": truthy(item.get("optional_install", False)),
                "featured_item": truthy(item.get("featured_item", False)),
                "package_selection": item.get("package_selection", "latest_eligible"),
            }
            if item.get("pinned_package_id"):
                body["pinned_package_id"] = int(item["pinned_package_id"])
            includes.append(body)
        return includes

    def label_id(self, client, assignment):
        label_id = assignment.get("label_id")
        if label_id:
            return label_id
        label_name = assignment.get("label_name")
        if not label_name:
            raise ProcessorError("assignments.includes entry requires label_id or label_name")
        return self.label_id_by_name(client, label_name)

    def label_id_by_name(self, client, label_name):
        label = find_exact(client, "/api/labels", "name", label_name)
        if not label:
            raise ProcessorError(f"Woodstar label not found: {label_name}")
        return label["id"]


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiAppUploader()
    PROCESSOR.execute_shell()
