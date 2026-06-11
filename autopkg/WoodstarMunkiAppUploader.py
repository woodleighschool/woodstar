#!/usr/local/autopkg/python

import os.path
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, find_exact  # noqa: E402

__all__ = ["WoodstarMunkiAppUploader"]

SUPPORTED_TARGET_ACTIONS = {
    "managed_installs",
    "managed_uninstalls",
    "managed_updates",
    "optional_installs",
    "featured_items",
    "default_installs",
}


class WoodstarMunkiAppUploader(Processor):
    description = "Upserts Woodstar Munki software, icon, and targets."

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
            "description": "Woodstar Munki software name. Defaults to pkginfo display_name, pkginfo name, or NAME.",
        },
        "description": {"required": False, "description": "Software description."},
        "category": {"required": False, "description": "Software category."},
        "developer": {"required": False, "description": "Software developer."},
        "icon_path": {
            "required": False,
            "description": "Icon file to upload and attach to the Munki software.",
        },
        "targets": {
            "required": False,
            "description": "Optional full targets object with include and exclude label entries.",
        },
    }
    output_variables = {
        "woodstar_software": {"description": "Munki software response."},
        "woodstar_software_id": {"description": "Woodstar Munki software ID."},
        "woodstar_icon_artifact": {"description": "Uploaded icon artifact response."},
        "woodstar_targets": {"description": "Munki software targets."},
        "woodstarmunkiappuploader_summary_result": {
            "description": "Summary of Woodstar app changes.",
        },
    }

    def main(self):
        self.env.pop("woodstarmunkiappuploader_summary_result", None)
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

        software, changed, action = self.upsert_software(client, pkginfo, name, icon_artifact)
        targets = software.get("targets") or empty_targets()

        self.env["woodstar_software"] = software
        self.env["woodstar_software_id"] = software["id"]
        self.env["woodstar_targets"] = targets
        if changed:
            self.env["woodstarmunkiappuploader_summary_result"] = {
                "summary_text": "Woodstar Munki app updated",
                "report_fields": ["id", "name", "action"],
                "data": {
                    "id": str(software["id"]),
                    "name": software["name"],
                    "action": action,
                },
            }
        self.output(f"{action} Woodstar Munki software {software['id']}: {software['name']}")

    def upsert_software(self, client, pkginfo, name, icon_artifact):
        body = {
            "name": name,
            "description": self.env.get("description") or pkginfo.get("description") or "",
            "category": self.env.get("category") or pkginfo.get("category") or "",
            "developer": self.env.get("developer") or pkginfo.get("developer") or "",
        }
        existing = find_exact(client, "/api/munki/software", "name", name)
        existing_detail = None
        if existing:
            existing_detail = client.get(f"/api/munki/software/{existing['id']}")
        if icon_artifact:
            body["icon_artifact_id"] = icon_artifact["id"]
        elif existing_detail and existing_detail.get("icon_artifact_id"):
            body["icon_artifact_id"] = existing_detail["icon_artifact_id"]
        body["targets"] = self.target_body(client, existing_detail)
        if existing:
            if self.software_matches(existing_detail, body):
                return existing_detail, False, "Skipped"
            self.output(f"Updating Woodstar Munki software: {name}")
            return client.put(f"/api/munki/software/{existing['id']}", body), True, "Updated"
        self.output(f"Creating Woodstar Munki software: {name}")
        return client.post("/api/munki/software", body), True, "Created"

    @staticmethod
    def software_matches(existing, body):
        for key, value in body.items():
            if normalized(existing.get(key)) != normalized(value):
                return False
        return True

    def software_name(self, pkginfo):
        return (
            self.env.get("name")
            or pkginfo.get("display_name")
            or pkginfo.get("name")
            or self.env.get("NAME")
        )

    def target_body(self, client, existing):
        target_config = self.env.get("targets")
        if not target_config:
            return (existing or {}).get("targets") or empty_targets()
        if not isinstance(target_config, dict):
            raise ProcessorError("targets must be a dictionary")
        include_inputs = target_config.get("include") or []
        if not isinstance(include_inputs, list):
            raise ProcessorError("targets.include must be a list")
        exclude_inputs = target_config.get("exclude") or []
        if not isinstance(exclude_inputs, list):
            raise ProcessorError("targets.exclude must be a list")

        include = [self.include_body(client, target) for target in include_inputs]
        exclude = self.exclude_body(client, exclude_inputs)
        return {"include": include, "exclude": exclude}

    def include_body(self, client, target):
        if not isinstance(target, dict):
            raise ProcessorError("targets.include entries must be dictionaries")
        if "priority" in target:
            raise ProcessorError("targets.include priority is not supported; order the include list instead")
        label_id = int(self.label_id(client, target, "targets.include entry"))
        body = {
            "label_id": label_id,
            "package": self.package_selector(target),
            "actions": self.actions(target),
        }
        return body

    @staticmethod
    def actions(target):
        actions = target.get("actions", ["managed_installs"])
        if not isinstance(actions, list) or not actions:
            raise ProcessorError("targets.include actions must be a non-empty list")
        for action in actions:
            if action not in SUPPORTED_TARGET_ACTIONS:
                raise ProcessorError(f"targets.include action {action!r} is not supported")
        return actions

    def package_selector(self, target):
        package = target.get("package") or {"strategy": "latest"}
        if not isinstance(package, dict):
            raise ProcessorError("targets.include package must be a dictionary")
        strategy = package.get("strategy", "latest")
        if strategy not in {"latest", "specific"}:
            raise ProcessorError("targets.include package.strategy must be latest or specific")
        if strategy == "latest":
            if package.get("package_id"):
                raise ProcessorError("targets.include latest package must not set package_id")
            return {"strategy": "latest"}
        package_id = package.get("package_id")
        if not package_id:
            raise ProcessorError("targets.include specific package requires package_id")
        return {"strategy": "specific", "package_id": int(package_id)}

    def exclude_body(self, client, inputs):
        values = [self.label_ref(client, target, "targets.exclude entry") for target in inputs]
        label_ids = [target["label_id"] for target in values]
        if len(label_ids) != len(set(label_ids)):
            raise ProcessorError("targets.exclude contains duplicate labels")
        return values

    def label_ref(self, client, target, context):
        if isinstance(target, (int, str)):
            return {"label_id": int(target)}
        if not isinstance(target, dict):
            raise ProcessorError(f"{context} must be a dictionary")
        return {"label_id": int(self.label_id(client, target, context))}

    def label_id(self, client, target, context):
        label_id = target.get("label_id")
        if label_id:
            return label_id
        label_name = target.get("label_name")
        if not label_name:
            raise ProcessorError(f"{context} requires label_id or label_name")
        return self.label_id_by_name(client, label_name)

    def label_id_by_name(self, client, label_name):
        label = find_exact(client, "/api/labels", "name", label_name)
        if not label:
            raise ProcessorError(f"Woodstar label not found: {label_name}")
        return label["id"]


def empty_targets():
    return {"include": [], "exclude": []}


def normalized(value):
    if isinstance(value, dict):
        return {key: normalized(item) for key, item in sorted(value.items()) if item is not None}
    if isinstance(value, list):
        return [normalized(item) for item in value]
    return value


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiAppUploader()
    PROCESSOR.execute_shell()
