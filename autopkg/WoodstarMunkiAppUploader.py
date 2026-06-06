#!/usr/local/autopkg/python

import os.path
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, find_exact, truthy  # noqa: E402

__all__ = ["WoodstarMunkiAppUploader"]


class WoodstarMunkiAppUploader(Processor):
    description = "Upserts a Woodstar Munki software title, icon, and assignments."

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
            "description": "Woodstar/Munki software name. Defaults to pkginfo name or NAME.",
        },
        "display_name": {
            "required": False,
            "description": "Display name. Defaults to pkginfo display_name or name.",
        },
        "description": {"required": False, "description": "Software description."},
        "category": {"required": False, "description": "Software category."},
        "developer": {"required": False, "description": "Software developer."},
        "icon_path": {
            "required": False,
            "description": "Icon file to upload and attach to the software title.",
        },
        "assign_to_all_devices": {
            "required": False,
            "description": "Create/update a required install assignment to All Hosts.",
            "default": True,
        },
        "assignment_label": {
            "required": False,
            "description": "Default assignment label name.",
            "default": "All Hosts",
        },
        "assignments": {
            "required": False,
            "description": "Optional list of assignment dictionaries.",
        },
    }
    output_variables = {
        "woodstar_software": {"description": "Software title response."},
        "woodstar_software_id": {"description": "Woodstar software title ID."},
        "woodstar_icon_artifact": {"description": "Uploaded icon artifact response."},
        "woodstar_assignments": {"description": "Created or updated assignments."},
        "woodstarmunkiappuploader_summary_result": {
            "description": "Summary of Woodstar app changes.",
        },
    }

    def main(self):
        client = client_from_env(self.env)
        pkginfo = self.env.get("munki_info") or self.env.get("pkginfo") or {}
        name = self.env.get("name") or pkginfo.get("name") or self.env.get("NAME")
        if not name:
            raise ProcessorError("name is required")

        icon_artifact = None
        icon_path = self.env.get("icon_path")
        if icon_path:
            icon_artifact = client.upload_artifact("icon", icon_path, os.path.basename(icon_path))
            self.env["woodstar_icon_artifact"] = icon_artifact

        software = self.upsert_software(client, pkginfo, name, icon_artifact)
        assignments = self.upsert_assignments(client, software)

        self.env["woodstar_software"] = software
        self.env["woodstar_software_id"] = software["id"]
        self.env["woodstar_assignments"] = assignments
        self.env["woodstarmunkiappuploader_summary_result"] = {
            "summary_text": "Woodstar Munki app processed",
            "report_fields": ["id", "name", "display_name", "assignments"],
            "data": {
                "id": str(software["id"]),
                "name": software["name"],
                "display_name": software.get("display_name", ""),
                "assignments": str(len(assignments)),
            },
        }
        self.output(f"Processed Woodstar software title {software['id']}: {software['name']}")

    def upsert_software(self, client, pkginfo, name, icon_artifact):
        body = {
            "name": name,
            "display_name": self.env.get("display_name") or pkginfo.get("display_name") or name,
            "description": self.env.get("description") or pkginfo.get("description") or "",
            "category": self.env.get("category") or pkginfo.get("category") or "",
            "developer": self.env.get("developer") or pkginfo.get("developer") or "",
        }
        if icon_artifact:
            body["icon_artifact_id"] = icon_artifact["id"]
        existing = find_exact(client, "/api/munki/software-titles", "name", name)
        if existing:
            self.output(f"Updating Woodstar software title: {name}")
            return client.patch(f"/api/munki/software-titles/{existing['id']}", body)
        self.output(f"Creating Woodstar software title: {name}")
        return client.post("/api/munki/software-titles", body)

    def upsert_assignments(self, client, software):
        assignment_inputs = self.env.get("assignments")
        if assignment_inputs is None and truthy(self.env.get("assign_to_all_devices", True)):
            assignment_inputs = [{"label_name": self.env.get("assignment_label") or "All Hosts"}]
        if not assignment_inputs:
            return []
        if not isinstance(assignment_inputs, list):
            raise ProcessorError("assignments must be a list")

        existing_page = client.get(
            "/api/munki/assignments",
            {"software_id": software["id"], "page_size": 1000, "sort": "priority.asc"},
        )
        existing = existing_page.get("items", [])
        out = []
        for index, assignment in enumerate(assignment_inputs, start=1):
            body = self.assignment_body(client, software["id"], assignment, existing, index)
            match = self.match_assignment(existing, body)
            if match:
                self.output(f"Updating assignment for label {body['label_id']}")
                out.append(client.patch(f"/api/munki/assignments/{match['id']}", body))
            else:
                self.output(f"Creating assignment for label {body['label_id']}")
                out.append(client.post("/api/munki/assignments", body))
        return out

    def assignment_body(self, client, software_id, assignment, existing, index):
        if not isinstance(assignment, dict):
            raise ProcessorError("assignment entries must be dictionaries")
        label_id = assignment.get("label_id")
        if not label_id:
            label_name = assignment.get("label_name") or assignment.get("label") or "All Hosts"
            label = find_exact(client, "/api/labels", "name", label_name)
            if not label:
                raise ProcessorError(f"Woodstar label not found: {label_name}")
            label_id = label["id"]

        effect = assignment.get("effect", "include")
        body = {
            "software_id": software_id,
            "priority": int(assignment.get("priority") or self.existing_priority(existing, label_id) or index),
            "label_id": int(label_id),
            "effect": effect,
        }
        if effect == "include":
            body.update(
                {
                    "action": assignment.get("action", "install"),
                    "optional_install": truthy(assignment.get("optional_install", False)),
                    "featured_item": truthy(assignment.get("featured_item", False)),
                    "package_selection": assignment.get("package_selection", "latest_eligible"),
                }
            )
            if assignment.get("pinned_package_id"):
                body["pinned_package_id"] = int(assignment["pinned_package_id"])
        return body

    @staticmethod
    def match_assignment(existing, body):
        for item in existing:
            if item.get("label_id") == body["label_id"] and item.get("effect") == body["effect"]:
                return item
        return None

    @staticmethod
    def existing_priority(existing, label_id):
        for item in existing:
            if item.get("label_id") == int(label_id):
                return item.get("priority")
        return None


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiAppUploader()
    PROCESSOR.execute_shell()
