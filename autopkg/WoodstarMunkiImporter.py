#!/usr/local/autopkg/python

import os.path
import plistlib
import subprocess
import sys
from datetime import datetime

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, truthy  # noqa: E402
from WoodstarLib.Munki import (  # noqa: E402
    PackageManager,
    PackageReferenceResolver,
    SoftwareManager,
    UNSET,
)

__all__ = ["WoodstarMunkiImporter"]

MAKEPKGINFO = "/usr/local/munki/makepkginfo"

UNSUPPORTED_INPUTS = {
    "extract_icon": "provide icon_path instead",
    "force_munkiimport": "use force to re-upload artifacts",
    "repo_subdirectory": "the direct importer does not write a Munki repository",
    "uninstaller_pkg_path": (
        "Woodstar cannot represent a separate uninstaller package artifact"
    ),
}


class WoodstarMunkiImporter(Processor):
    description = (
        "Generates Munki pkginfo and uploads the software, package, installer, "
        "optional icon, and targets directly to Woodstar."
    )

    input_variables = {
        "WOODSTAR_URL": {
            "required": True,
            "description": (
                "Woodstar HTTPS origin, for example https://woodstar.example."
            ),
        },
        "WOODSTAR_API_KEY": {
            "required": True,
            "description": "Woodstar admin API key.",
        },
        "WOODSTAR_CA_FILE": {
            "required": False,
            "description": "PEM CA file for a private Woodstar certificate chain.",
        },
        "pkg_path": {
            "required": True,
            "description": "Local package or disk image to inspect and upload.",
        },
        "pkginfo": {
            "required": False,
            "description": "Dictionary of keys to merge into generated Munki pkginfo.",
        },
        "munkiimport_pkgname": {
            "required": False,
            "description": "Package name passed to makepkginfo --pkgname.",
        },
        "munkiimport_appname": {
            "required": False,
            "description": "Application name passed to makepkginfo --appname.",
        },
        "additional_makepkginfo_options": {
            "required": False,
            "description": "Additional makepkginfo command-line options.",
        },
        "version_comparison_key": {
            "required": False,
            "description": "Version key to set on every generated installs item.",
        },
        "metadata_additions": {
            "required": False,
            "description": "Dictionary merged into the generated _metadata.",
        },
        "icon_path": {
            "required": False,
            "description": "Local icon file to upload and attach to the software.",
        },
        "targets": {
            "required": False,
            "description": "Full targets object with include and exclude label entries.",
        },
        "force": {
            "required": False,
            "description": "Upload artifacts even when their metadata is unchanged.",
            "default": False,
        },
    }
    output_variables = {
        "munki_info": {"description": "Generated Munki pkginfo dictionary."},
        "woodstar_software": {"description": "Woodstar Munki software response."},
        "woodstar_software_id": {"description": "Woodstar Munki software ID."},
        "woodstar_targets": {"description": "Resolved Munki software targets."},
        "woodstar_package": {"description": "Woodstar Munki package response."},
        "woodstar_package_id": {"description": "Woodstar Munki package ID."},
        "woodstarmunkiimporter_summary_result": {
            "description": "Summary of direct Woodstar changes.",
        },
    }

    def main(self):
        self.env.pop("woodstarmunkiimporter_summary_result", None)
        self.reject_unsupported_inputs()
        pkg_path, icon_path = self.local_paths()
        pkginfo = self.generate_pkginfo(pkg_path)
        client = client_from_env(self.env)
        force = truthy(self.env.get("force", False))

        software_manager = SoftwareManager(client, self.output)
        package_manager = PackageManager(client, self.output)
        targets = self.env["targets"] if "targets" in self.env else UNSET

        name, existing_software, software_body = software_manager.prepare(
            pkginfo,
            targets=targets,
        )
        resolver = PackageReferenceResolver.from_client(client)
        provisional_software_id = (
            int(existing_software["id"]) if existing_software else 0
        )
        package_body = package_manager.package_body(
            pkginfo,
            provisional_software_id,
            resolver,
        )
        installer_path = package_manager.installer_artifact_path(pkginfo, pkg_path)

        software, software_action, software_changed = software_manager.save(
            name,
            existing_software,
            software_body,
        )
        software, icon_uploaded = software_manager.attach_icon(
            software,
            icon_path,
            force,
        )
        if icon_uploaded and software_action == "Skipped":
            software_action = "Updated"

        software_id = int(software["id"])
        package_body["software_id"] = software_id
        package, package_action, package_changed, installer_uploaded = (
            package_manager.save_package(
                software_id,
                package_body,
                installer_path,
                force,
            )
        )

        self.env["munki_info"] = pkginfo
        self.env["woodstar_software"] = software
        self.env["woodstar_software_id"] = software_id
        self.env["woodstar_targets"] = software["targets"]
        self.env["woodstar_package"] = package
        self.env["woodstar_package_id"] = package["id"]

        changed = software_changed or icon_uploaded or package_changed or installer_uploaded
        if changed:
            self.env["woodstarmunkiimporter_summary_result"] = {
                "summary_text": "Woodstar Munki item updated",
                "report_fields": [
                    "software_id",
                    "package_id",
                    "name",
                    "version",
                    "software_action",
                    "package_action",
                    "icon_uploaded",
                    "installer_uploaded",
                ],
                "data": {
                    "software_id": str(software_id),
                    "package_id": str(package["id"]),
                    "name": name,
                    "version": package["version"],
                    "software_action": software_action,
                    "package_action": package_action,
                    "icon_uploaded": str(icon_uploaded),
                    "installer_uploaded": str(installer_uploaded),
                },
            }

        self.output(
            f"{software_action} Woodstar Munki software {software_id}: {name}"
        )
        self.output(
            f"{package_action} Woodstar package {package['id']}: "
            f"{name} {package['version']}"
        )

    def reject_unsupported_inputs(self):
        for key, guidance in UNSUPPORTED_INPUTS.items():
            if self.env.get(key) not in (None, False, ""):
                raise ProcessorError(f"{key} is not supported: {guidance}")

    def local_paths(self):
        pkg_path = self.local_file("pkg_path", required=True)
        icon_path = self.local_file("icon_path")
        return pkg_path, icon_path

    def local_file(self, key, required=False):
        value = self.env.get(key)
        if not value:
            if required:
                raise ProcessorError(f"{key} is required")
            return None
        path = os.path.abspath(str(value))
        if not os.path.isfile(path):
            raise ProcessorError(f"{key} does not exist: {path}")
        return path

    def generate_pkginfo(self, pkg_path):
        args = [MAKEPKGINFO, pkg_path]
        self.append_option(args, "munkiimport_pkgname", "--pkgname")
        self.append_option(args, "munkiimport_appname", "--appname")

        additional_options = self.env.get("additional_makepkginfo_options", [])
        if not isinstance(additional_options, list) or not all(
            isinstance(option, str) for option in additional_options
        ):
            raise ProcessorError(
                "additional_makepkginfo_options must be a list of strings"
            )
        args.extend(additional_options)

        try:
            result = subprocess.run(
                args,
                capture_output=True,
                check=False,
            )
        except OSError as err:
            raise ProcessorError(
                f"makepkginfo execution failed with error code {err.errno}: "
                f"{err.strerror}"
            ) from err

        stderr = result.stderr.decode(errors="replace")
        for line in stderr.splitlines():
            self.output(line)
        if result.returncode != 0:
            raise ProcessorError(
                f"creating pkginfo for {pkg_path} failed: {stderr.strip()}"
            )
        try:
            pkginfo = plistlib.loads(result.stdout)
        except (plistlib.InvalidFileException, ValueError) as err:
            raise ProcessorError(
                f"makepkginfo returned invalid plist for {pkg_path}: {err}"
            ) from err
        return self.apply_pkginfo_inputs(pkginfo)

    def append_option(self, args, env_key, command_option):
        value = self.env.get(env_key)
        if value is None or value == "":
            return
        if not isinstance(value, str):
            raise ProcessorError(f"{env_key} must be a string")
        args.extend([command_option, value])

    def apply_pkginfo_inputs(self, pkginfo):
        overrides = self.env.get("pkginfo", {})
        if not isinstance(overrides, dict):
            raise ProcessorError("pkginfo must be a dictionary")
        for key, value in overrides.items():
            if (
                key == "force_install_after_date"
                and isinstance(value, str)
                and value.endswith("Z")
            ):
                try:
                    value = datetime.strptime(value[:-1], "%Y-%m-%dT%H:%M:%S")
                except ValueError:
                    pass
            pkginfo[key] = value

        metadata_additions = self.env.get("metadata_additions", {})
        if not isinstance(metadata_additions, dict):
            raise ProcessorError("metadata_additions must be a dictionary")
        metadata = pkginfo.get("_metadata")
        if not isinstance(metadata, dict):
            raise ProcessorError("generated pkginfo _metadata must be a dictionary")
        metadata.update(metadata_additions)

        comparison_key = self.env.get("version_comparison_key")
        if comparison_key:
            if not isinstance(comparison_key, str):
                raise ProcessorError("version_comparison_key must be a string")
            for item in pkginfo.get("installs", []):
                if comparison_key not in item:
                    raise ProcessorError(
                        f"version_comparison_key {comparison_key!r} could not be "
                        f"found in the installs item for path {item.get('path')!r}"
                    )
                item["version_comparison_key"] = comparison_key
        return pkginfo


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiImporter()
    PROCESSOR.execute_shell()
