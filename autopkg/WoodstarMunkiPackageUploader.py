#!/usr/local/autopkg/python

import os.path
import plistlib
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, list_items, needs_object, truthy  # noqa: E402

__all__ = ["WoodstarMunkiPackageUploader"]

PACKAGE_DIRECT_KEYS = (
    "unattended_install",
    "unattended_uninstall",
    "minimum_munki_version",
    "minimum_os_version",
    "maximum_os_version",
    "supported_architectures",
    "blocking_applications",
    "blocking_applications_none",
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

PACKAGE_DEFAULTS = {
    "installer_type": "pkg",
    "unattended_install": False,
    "unattended_uninstall": False,
    "uninstall_method": "",
    "restart_action": "",
    "minimum_munki_version": "",
    "minimum_os_version": "",
    "maximum_os_version": "",
    "supported_architectures": [],
    "blocking_applications": [],
    "blocking_applications_none": False,
    "installable_condition": "",
    "blocking_applications_manual_quit_only": False,
    "blocking_applications_quit_script": "",
    "requires": [],
    "update_for": [],
    "on_demand": False,
    "precache": False,
    "autoremove": False,
    "apple_item": False,
    "suppress_bundle_relocation": False,
    "force_install_after_date": None,
    "installed_size": 0,
    "package_path": "",
    "installer_choices_xml": [],
    "installer_environment": [],
    "installs": [],
    "receipts": [],
    "items_to_copy": [],
    "notes": "",
    "installcheck_script": "",
    "uninstallcheck_script": "",
    "preinstall_script": "",
    "postinstall_script": "",
    "preuninstall_script": "",
    "postuninstall_script": "",
    "uninstall_script": "",
    "version_script": "",
    "preinstall_alert": {"enabled": False},
    "preuninstall_alert": {"enabled": False},
}


class WoodstarMunkiPackageUploader(Processor):
    description = "Uploads Munki package artifacts and creates or updates a Woodstar package."

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
        "MUNKI_REPO": {
            "required": True,
            "description": "Munki repo path containing MunkiImporter output.",
        },
        "pkginfo_repo_path": {
            "required": False,
            "description": "MunkiImporter pkginfo output path. Empty when MunkiImporter reuses an existing item.",
        },
        "pkg_repo_path": {
            "required": False,
            "description": "MunkiImporter package artifact path. Required for package-bearing pkginfo.",
        },
        "software_id": {
            "required": False,
            "description": "Existing Woodstar Munki software ID. Defaults to woodstar_software_id.",
        },
        "force": {
            "required": False,
            "description": "Upload package artifacts even when Woodstar already has the same file.",
            "default": False,
        },
    }
    output_variables = {
        "woodstar_package": {"description": "Woodstar package response."},
        "woodstar_package_id": {"description": "Woodstar package ID."},
        "woodstarmunkipackageuploader_summary_result": {
            "description": "Summary of Woodstar package upload.",
        },
    }

    def main(self):
        self.env.pop("woodstarmunkipackageuploader_summary_result", None)
        client = client_from_env(self.env)
        pkginfo = self.pkginfo()
        software_id = self.software_id()
        installer_path = self.installer_artifact_path(pkginfo)
        force = truthy(self.env.get("force", False))

        resolver = PackageReferenceResolver.from_client(client)
        body = self.package_body(pkginfo, software_id, resolver)
        package, action, package_changed, installer_uploaded = self.save_package(
            client,
            software_id,
            body,
            installer_path,
            force,
        )

        self.env["woodstar_package"] = package
        self.env["woodstar_package_id"] = package["id"]
        if package_changed or installer_uploaded:
            self.env["woodstarmunkipackageuploader_summary_result"] = {
                "summary_text": "Woodstar Munki package updated",
                "report_fields": [
                    "id",
                    "software",
                    "version",
                    "action",
                    "package_uploaded",
                ],
                "data": {
                    "id": str(package["id"]),
                    "software": package.get("software_name", ""),
                    "version": package["version"],
                    "action": action,
                    "package_uploaded": str(installer_uploaded),
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
            return None
        if not os.path.isfile(path):
            raise ProcessorError(f"MunkiImporter pkginfo not found: {path}")
        return path

    def pkginfo(self):
        pkginfo_path = self.repo_pkginfo_path()
        if not pkginfo_path:
            pkginfo_path = self.existing_repo_pkginfo_path()
            self.output(f"Using existing Munki pkginfo: {pkginfo_path}")
        else:
            self.output(f"Using MunkiImporter pkginfo: {pkginfo_path}")
        try:
            return self.load_pkginfo(pkginfo_path)
        except (OSError, plistlib.InvalidFileException, ValueError) as err:
            raise ProcessorError(f"failed to read Munki pkginfo {pkginfo_path}: {err}") from err

    def existing_repo_pkginfo_path(self):
        installer_location = self.installer_item_location()
        pkginfos_dir = os.path.join(str(self.env.get("MUNKI_REPO") or ""), "pkgsinfo")
        if not os.path.isdir(pkginfos_dir):
            raise ProcessorError("MUNKI_REPO pkgsinfo directory was not found")
        matches = []
        for root, _dirs, filenames in os.walk(pkginfos_dir):
            for filename in filenames:
                path = os.path.join(root, filename)
                if self.pkginfo_installer_location(path) == installer_location:
                    matches.append(path)
        if not matches:
            raise ProcessorError(
                "pkginfo_repo_path was empty from MunkiImporter and no existing "
                f"pkginfo matched installer_item_location {installer_location}"
            )
        return sorted(matches)[0]

    def pkginfo_installer_location(self, path):
        try:
            pkginfo = self.load_pkginfo(path)
        except (OSError, plistlib.InvalidFileException, ValueError):
            return None
        return pkginfo.get("installer_item_location")

    def installer_item_location(self):
        package_path = self.repo_path(self.env.get("pkg_repo_path"))
        if not package_path:
            raise ProcessorError("pkg_repo_path is required when pkginfo_repo_path is empty")
        munki_repo = self.env.get("MUNKI_REPO")
        if not munki_repo:
            raise ProcessorError("MUNKI_REPO is required when pkginfo_repo_path is empty")
        package_path = os.path.abspath(package_path)
        pkgs_dir = os.path.abspath(os.path.join(str(munki_repo), "pkgs"))
        try:
            if os.path.commonpath([pkgs_dir, package_path]) != pkgs_dir:
                raise ValueError
            return os.path.relpath(package_path, pkgs_dir)
        except ValueError as err:
            raise ProcessorError(f"pkg_repo_path is not inside MUNKI_REPO pkgs: {package_path}") from err

    def package_path(self):
        path = self.repo_path(self.env.get("pkg_repo_path"))
        if path and os.path.isfile(path):
            return path
        raise ProcessorError("pkg_repo_path is required for package-bearing MunkiImporter pkginfo")

    def installer_artifact_path(self, pkginfo):
        if self.needs_installer_artifact(pkginfo):
            return self.package_path()
        return None

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

    def base_package_body(self, pkginfo, software_id):
        body = self.package_mutation_from_pkginfo(pkginfo)
        body["software_id"] = software_id
        return body

    def package_body(self, pkginfo, software_id, resolver):
        body = self.base_package_body(pkginfo, software_id)
        body["requires"] = resolver.resolve(pkginfo.get("requires", []), "requires")
        body["update_for"] = resolver.resolve(pkginfo.get("update_for", []), "update_for")
        return body

    def package_mutation_from_pkginfo(self, pkginfo):
        if not isinstance(pkginfo, dict):
            raise ProcessorError("pkginfo must be a dictionary")
        version = pkginfo.get("version")
        if not isinstance(version, str) or version == "":
            raise ProcessorError("pkginfo version is required")

        body = default_package_mutation()
        body["version"] = version
        body["installer_type"] = self.installer_type(pkginfo)
        uninstall_method = self.uninstall_method(pkginfo)
        if uninstall_method:
            body["uninstall_method"] = uninstall_method
        for key in PACKAGE_DIRECT_KEYS:
            if key in pkginfo:
                body[key] = pkginfo[key]
        if "blocking_applications" in pkginfo:
            body["blocking_applications_none"] = len(pkginfo["blocking_applications"]) == 0
        force_date = body.get("force_install_after_date")
        if hasattr(force_date, "isoformat"):
            body["force_install_after_date"] = force_date.isoformat()
        if "RestartAction" in pkginfo:
            body["restart_action"] = pkginfo["RestartAction"]
        if "OnDemand" in pkginfo:
            body["on_demand"] = pkginfo["OnDemand"]
        if "installer_environment" in pkginfo:
            body["installer_environment"] = self.installer_environment(pkginfo["installer_environment"])
        if "installer_choices_xml" in pkginfo:
            body["installer_choices_xml"] = self.installer_choices_xml(pkginfo["installer_choices_xml"])
        if "installs" in pkginfo:
            body["installs"] = self.install_items(pkginfo["installs"])
        if "receipts" in pkginfo:
            body["receipts"] = self.receipts(pkginfo["receipts"])
        for munki_key, woodstar_key in (
            ("preinstall_alert", "preinstall_alert"),
            ("preuninstall_alert", "preuninstall_alert"),
        ):
            if munki_key in pkginfo:
                body[woodstar_key] = self.alert(pkginfo[munki_key], munki_key)
        return body

    def save_package(self, client, software_id, body, installer_path, force=False):
        existing = self.existing_package(client, software_id, body["version"])
        if existing:
            existing = client.get(f"/api/munki/packages/{existing['id']}")
        installer_object_id = None
        installer_uploaded = False
        if body["installer_type"] != "nopkg":
            if not installer_path:
                raise ProcessorError("package-bearing pkginfo requires an installer artifact")
            if not existing or needs_object(existing, "installer", installer_path, force):
                installer = client.upload_package_installer(installer_path)
                installer_object_id = installer["id"]
                installer_uploaded = True
            else:
                installer_object_id = existing["installer_object_id"]
            body = {**body, "installer_object_id": installer_object_id}

        if existing and self.package_matches(existing, body):
            return existing, "Skipped", False, installer_uploaded

        try:
            if existing:
                update_body = mutation_request_body(body)
                del update_body["software_id"]
                saved = client.put(f"/api/munki/packages/{existing['id']}", update_body)
                return saved, "Updated", True, installer_uploaded
            saved = client.post("/api/munki/packages", mutation_request_body(body))
            return saved, "Created", True, installer_uploaded
        except ProcessorError:
            if installer_uploaded:
                try:
                    client.delete(f"/api/munki/package-installers/{installer_object_id}")
                except ProcessorError:
                    pass
            raise

    @staticmethod
    def existing_package(client, software_id, version):
        for item in list_items(
            client,
            "/api/munki/packages",
            {"software_id": software_id, "q": version, "per_page": 1000},
        ):
            if item.get("software_id") == software_id and item.get("version") == version:
                return item
        return None

    @staticmethod
    def package_matches(existing, body):
        for key, value in body.items():
            if key in {"requires", "update_for"}:
                if reference_ids(existing.get(key)) != reference_ids(value):
                    return False
                continue
            if normalized(existing.get(key)) != normalized(value):
                return False
        return True

    @staticmethod
    def installer_type(pkginfo):
        value = pkginfo.get("installer_type", "pkg")
        if not isinstance(value, str):
            raise ProcessorError("pkginfo installer_type must be a string")
        return value

    @staticmethod
    def uninstall_method(pkginfo):
        value = pkginfo.get("uninstall_method", "")
        if not isinstance(value, str):
            raise ProcessorError("pkginfo uninstall_method must be a string")
        return value

    @staticmethod
    def installer_environment(value):
        if not isinstance(value, dict):
            raise ProcessorError("pkginfo installer_environment must be a dictionary")
        return [{"name": name, "value": value[name]} for name in sorted(value)]

    @staticmethod
    def installer_choices_xml(values):
        if not isinstance(values, list):
            raise ProcessorError("pkginfo installer_choices_xml must be a list")
        choices = []
        for value in values:
            if not isinstance(value, dict):
                raise ProcessorError("pkginfo installer_choices_xml entries must be dictionaries")
            choice_identifier = value.get("choiceIdentifier")
            if not isinstance(choice_identifier, str) or not choice_identifier.strip():
                raise ProcessorError(
                    "pkginfo installer_choices_xml choiceIdentifier must be a non-empty string"
                )
            choice_attribute = value.get("choiceAttribute", "")
            if not isinstance(choice_attribute, str):
                raise ProcessorError(
                    "pkginfo installer_choices_xml choiceAttribute must be a string"
                )
            if "attributeSetting" not in value:
                raise ProcessorError(
                    "pkginfo installer_choices_xml attributeSetting is required"
                )
            try:
                attribute_setting = int(value["attributeSetting"])
            except (TypeError, ValueError) as err:
                raise ProcessorError("pkginfo installer_choices_xml attributeSetting must be an integer") from err
            choices.append(
                {
                    "choice_identifier": choice_identifier,
                    "choice_attribute": choice_attribute,
                    "attribute_setting": attribute_setting,
                }
            )
        return choices

    @staticmethod
    def install_items(values):
        if not isinstance(values, list):
            raise ProcessorError("pkginfo installs must be a list")
        items = []
        for value in values:
            if not isinstance(value, dict):
                raise ProcessorError("pkginfo installs entries must be dictionaries")
            item = {
                "type": value.get("type", "file"),
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


def default_package_mutation():
    return {
        key: [*value] if isinstance(value, list) else dict(value) if isinstance(value, dict) else value
        for key, value in PACKAGE_DEFAULTS.items()
    }


def mutation_request_body(body):
    request_body = {key: value for key, value in body.items() if value is not None}
    for key in ("restart_action", "uninstall_method"):
        if request_body.get(key) == "":
            del request_body[key]
    return request_body


def normalized(value):
    if value is None:
        return ""
    if isinstance(value, dict):
        return {key: normalized(item) for key, item in sorted(value.items()) if item is not None}
    if isinstance(value, list):
        return [normalized(item) for item in value]
    if hasattr(value, "isoformat"):
        return value.isoformat()
    return value


def reference_ids(values):
    return [
        (item.get("software_id"), item.get("package_id") or 0)
        for item in values or []
    ]


def reference_request(values):
    return [
        {
            "software_id": item["software_id"],
            **({"package_id": item["package_id"]} if item.get("package_id") else {}),
        }
        for item in values or []
    ]


class PackageReferenceResolver:
    def __init__(self, software, packages):
        self.software = {}
        self.versioned = {}
        for item in software:
            self.software.setdefault(str(item["name"]), []).append(item)
        for package in packages:
            key = (
                str(package["software_name"]),
                munki_version(str(package["version"])),
            )
            self.versioned.setdefault(key, []).append(package)

    @classmethod
    def from_client(cls, client):
        return cls(
            list_items(client, "/api/munki/software"),
            list_items(client, "/api/munki/packages"),
        )

    def resolve(self, values, key):
        if not isinstance(values, list):
            raise ProcessorError(f"pkginfo {key} must be a list")
        return [self.resolve_one(value, key) for value in values]

    def resolve_one(self, value, key):
        if not isinstance(value, str) or not value.strip():
            raise ProcessorError(f"pkginfo {key} entries must be Munki item names")
        value = value.strip()
        name, version = split_munki_name(value)
        if version:
            candidates = [
                {
                    "software_id": package["software_id"],
                    "package_id": package["id"],
                }
                for package in self.versioned.get((name, munki_version(version)), [])
            ]
        else:
            candidates = [
                {"software_id": software["id"]}
                for software in self.software.get(name, [])
            ]
        if not candidates:
            raise ProcessorError(f"pkginfo {key} item {value!r} was not found in Woodstar")
        unique = {tuple(sorted(candidate.items())) for candidate in candidates}
        if len(unique) != 1:
            raise ProcessorError(f"pkginfo {key} item {value!r} is ambiguous in Woodstar")
        return candidates[0]


def split_munki_name(value):
    for separator in ("--", "-"):
        if separator in value:
            chunks = value.split(separator)
            version = chunks.pop()
            name = separator.join(chunks)
            if version and version[0].isdigit():
                return name, version
    return value, ""


def munki_version(value):
    parts = value.split(".")
    while len(parts) > 2 and parts[-1] == "0":
        parts.pop()
    return ".".join(parts)


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiPackageUploader()
    PROCESSOR.execute_shell()
