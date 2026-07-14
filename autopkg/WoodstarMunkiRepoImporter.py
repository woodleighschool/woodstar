#!/usr/local/autopkg/python

import os
import plistlib
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import client_from_env, find_exact, list_items, truthy  # noqa: E402
from WoodstarMunkiAppUploader import WoodstarMunkiAppUploader  # noqa: E402
from WoodstarMunkiPackageUploader import (  # noqa: E402
    PackageReferenceResolver,
    WoodstarMunkiPackageUploader,
    reference_request,
)

__all__ = ["WoodstarMunkiRepoImporter"]


class WoodstarMunkiRepoImporter(Processor):
    description = "Imports a Munki repo's pkginfo items into Woodstar."

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
            "description": "Munki repo path containing pkgsinfo, pkgs, and icons.",
        },
        "force": {
            "required": False,
            "description": "Upload package artifacts even when Woodstar already has the same file.",
            "default": False,
        },
    }
    output_variables = {
        "woodstar_imported_pkginfo_count": {"description": "Number of pkginfo files processed."},
        "woodstarmunkirepoimporter_summary_result": {
            "description": "Summary of Woodstar repo import changes.",
        },
    }

    def main(self):
        self.env.pop("woodstarmunkirepoimporter_summary_result", None)
        client = client_from_env(self.env)
        munki_repo = self.munki_repo()
        force = truthy(self.env.get("force", False))

        counts = import_counts()

        entries, skipped_duplicates = pkginfo_entries(munki_repo)
        counts["duplicate_pkginfos_skipped"] = skipped_duplicates
        if skipped_duplicates:
            self.output(f"Skipping {skipped_duplicates} duplicate Munki pkginfo item(s)")

        self.preflight_entries(client, munki_repo, entries)
        prepared = [
            self.prepare_pkginfo(client, munki_repo, pkginfo_path, pkginfo, counts)
            for pkginfo_path, pkginfo in entries
        ]
        resolver = PackageReferenceResolver.from_client(client)
        for item in prepared:
            self.finish_pkginfo(client, item, resolver, force, counts)

        self.env["woodstar_imported_pkginfo_count"] = counts["pkginfos"]
        if changed_count(counts):
            self.env["woodstarmunkirepoimporter_summary_result"] = {
                "summary_text": "Woodstar Munki repo imported",
                "report_fields": [
                    "pkginfos",
                    "software_created",
                    "packages_created",
                    "packages_updated",
                    "duplicate_pkginfos_skipped",
                    "package_binaries_uploaded",
                ],
                "data": {
                    "pkginfos": str(counts["pkginfos"]),
                    "software_created": str(counts["software_created"]),
                    "packages_created": str(counts["packages_created"]),
                    "packages_updated": str(counts["packages_updated"]),
                    "duplicate_pkginfos_skipped": str(counts["duplicate_pkginfos_skipped"]),
                    "package_binaries_uploaded": str(counts["package_binaries_uploaded"]),
                },
            }
        self.output(f"Processed {counts['pkginfos']} Munki pkginfo item(s)")

    def preflight_entries(self, client, munki_repo, entries):
        self.validate_software_metadata(entries)
        software = list_items(client, "/api/munki/software")
        packages = list_items(client, "/api/munki/packages")
        software_by_name = {item["name"]: item for item in software}
        package_keys = {
            (item["software_name"], item["version"])
            for item in packages
        }
        next_synthetic_id = -1

        validated = []
        for pkginfo_path, pkginfo in entries:
            app = WoodstarMunkiAppUploader()
            app.env = {}
            name = app.software_name(pkginfo)
            if not isinstance(name, str) or not name.strip():
                raise ProcessorError(f"name is required for pkginfo {pkginfo_path}")
            name = name.strip()

            software_item = software_by_name.get(name)
            if not software_item:
                software_item = {"id": next_synthetic_id, "name": name}
                next_synthetic_id -= 1
                software.append(software_item)
                software_by_name[name] = software_item

            package = WoodstarMunkiPackageUploader()
            package.env = {
                "MUNKI_REPO": munki_repo,
                "pkginfo_repo_path": pkginfo_path,
                "software_id": software_item["id"],
            }
            installer_path = repo_package_path(munki_repo, pkginfo.get("installer_item_location"))
            if installer_path:
                package.env["pkg_repo_path"] = installer_path
            body = package.base_package_body(pkginfo, software_item["id"])
            package.installer_artifact_path(pkginfo)

            package_key = (name, body["version"])
            if package_key not in package_keys:
                packages.append(
                    {
                        "id": next_synthetic_id,
                        "software_id": software_item["id"],
                        "software_name": name,
                        "version": body["version"],
                    }
                )
                next_synthetic_id -= 1
                package_keys.add(package_key)
            validated.append((pkginfo_path, pkginfo))

        resolver = PackageReferenceResolver(software, packages)
        for pkginfo_path, pkginfo in validated:
            resolver.resolve(pkginfo.get("requires", []), f"requires in {pkginfo_path}")
            resolver.resolve(pkginfo.get("update_for", []), f"update_for in {pkginfo_path}")

    @staticmethod
    def validate_software_metadata(entries):
        by_name = {}
        for pkginfo_path, pkginfo in entries:
            name = pkginfo.get("name")
            if not isinstance(name, str) or not name.strip():
                raise ProcessorError(f"name is required for pkginfo {pkginfo_path}")
            name = name.strip()
            metadata = {}
            for key in ("display_name", "description", "category", "developer"):
                value = pkginfo.get(key, "")
                if not isinstance(value, str):
                    raise ProcessorError(f"{key} must be a string in pkginfo {pkginfo_path}")
                metadata[key] = value
            if metadata["display_name"].strip() == name:
                metadata["display_name"] = ""
            existing = by_name.get(name)
            if existing is not None and existing != metadata:
                raise ProcessorError(
                    f"pkginfo metadata differs across versions of {name}: {pkginfo_path}"
                )
            by_name[name] = metadata

    def munki_repo(self):
        munki_repo = self.env.get("MUNKI_REPO")
        if not munki_repo:
            raise ProcessorError("MUNKI_REPO is required")
        munki_repo = os.path.abspath(str(munki_repo))
        if not os.path.isdir(os.path.join(munki_repo, "pkgsinfo")):
            raise ProcessorError(f"MUNKI_REPO pkgsinfo directory was not found: {munki_repo}")
        return munki_repo

    def prepare_pkginfo(self, client, munki_repo, pkginfo_path, pkginfo, counts):
        counts["pkginfos"] += 1
        app = WoodstarMunkiAppUploader()
        app.env = {
            "WOODSTAR_URL": self.env.get("WOODSTAR_URL"),
            "WOODSTAR_API_KEY": self.env.get("WOODSTAR_API_KEY"),
            "WOODSTAR_CA_FILE": self.env.get("WOODSTAR_CA_FILE"),
        }
        app.env["pkginfo"] = pkginfo

        name = app.software_name(pkginfo)
        if not name:
            raise ProcessorError(f"name is required for pkginfo {pkginfo_path}")
        self.output(f"Importing Munki pkginfo: {name} {pkginfo.get('version') or ''}")
        software, software_created = self.software_for_pkginfo(client, munki_repo, pkginfo, name)
        if software_created:
            counts["software_created"] += 1

        package = WoodstarMunkiPackageUploader()
        package.env = {
            "WOODSTAR_URL": self.env.get("WOODSTAR_URL"),
            "WOODSTAR_API_KEY": self.env.get("WOODSTAR_API_KEY"),
            "WOODSTAR_CA_FILE": self.env.get("WOODSTAR_CA_FILE"),
            "MUNKI_REPO": munki_repo,
            "software_id": software["id"],
            "pkginfo_repo_path": pkginfo_path,
        }
        installer_path = repo_package_path(munki_repo, pkginfo.get("installer_item_location"))
        if installer_path:
            package.env["pkg_repo_path"] = installer_path

        body = package.base_package_body(pkginfo, int(software["id"]))
        existing = package.existing_package(client, int(software["id"]), body["version"])
        if existing:
            existing = client.get(f"/api/munki/packages/{existing['id']}")
            body["requires"] = reference_request(existing.get("requires"))
            body["update_for"] = reference_request(existing.get("update_for"))
        saved, package_action, package_changed = package.save_package(client, int(software["id"]), body)

        return {
            "pkginfo": pkginfo,
            "package": package,
            "software_id": int(software["id"]),
            "saved": saved,
            "initial_action": package_action,
            "changed": package_changed,
        }

    def finish_pkginfo(self, client, item, resolver, force, counts):
        package = item["package"]
        pkginfo = item["pkginfo"]
        body = package.package_body(pkginfo, item["software_id"], resolver)
        saved, package_action, package_changed = package.save_package(
            client,
            item["software_id"],
            body,
        )
        changed = item["changed"] or package_changed
        action = (
            item["initial_action"]
            if item["initial_action"] != "Skipped"
            else package_action
        )
        if changed:
            increment_action(counts, "packages", action)

        installer_path = package.installer_artifact_path(pkginfo)
        installer_uploaded = package.attach_binary(client, saved, "installer", installer_path, force)
        counts["package_binaries_uploaded"] += int(installer_uploaded)

    def software_for_pkginfo(self, client, munki_repo, pkginfo, name):
        existing = find_exact(client, "/api/munki/software", "name", name)
        if existing:
            return client.get(f"/api/munki/software/{existing['id']}"), False

        body = {
            "name": name,
            "display_name": pkginfo.get("display_name", ""),
            "description": pkginfo.get("description", ""),
            "category": pkginfo.get("category", ""),
            "developer": pkginfo.get("developer", ""),
            "targets": {"include": [], "exclude": []},
        }
        if body["display_name"].strip() == name:
            body["display_name"] = ""
        self.output(f"Creating Woodstar Munki software: {name}")
        software = client.post("/api/munki/software", body)

        icon_path = repo_icon_path(munki_repo, pkginfo)
        if icon_path:
            client.attach_object(f"/api/munki/software/{software['id']}/icon", icon_path)
            software = client.get(f"/api/munki/software/{software['id']}")
        return software, True


def pkginfo_entries(munki_repo):
    entries = []
    skipped = 0
    selected = {}
    for path in all_pkginfo_paths(munki_repo):
        pkginfo = load_pkginfo(path)
        key = pkginfo_key(pkginfo)
        if not key:
            entries.append((path, pkginfo))
            continue
        rank = pkginfo_rank(path, pkginfo)
        current = selected.get(key)
        if current and current[0] >= rank:
            skipped += 1
            continue
        if current:
            skipped += 1
        selected[key] = (rank, path, pkginfo)
    entries.extend((path, pkginfo) for _rank, path, pkginfo in selected.values())
    entries.sort(key=lambda entry: entry[0])
    return entries, skipped


def all_pkginfo_paths(munki_repo):
    pkgsinfo_dir = os.path.join(munki_repo, "pkgsinfo")
    paths = []
    for root, _dirs, filenames in os.walk(pkgsinfo_dir):
        for filename in filenames:
            if filename.startswith("."):
                continue
            paths.append(os.path.join(root, filename))
    return sorted(paths)


def pkginfo_key(pkginfo):
    name = pkginfo.get("name")
    version = pkginfo.get("version")
    if not name or not version:
        return None
    return str(name), str(version)


def pkginfo_rank(path, pkginfo):
    created_at = ((pkginfo.get("_metadata") or {}).get("creation_date") or "")
    if hasattr(created_at, "isoformat"):
        created_at = created_at.isoformat()
    return str(created_at), os.path.getmtime(path), path


def load_pkginfo(path):
    try:
        with open(path, "rb") as handle:
            return plistlib.load(handle)
    except (OSError, plistlib.InvalidFileException, ValueError) as err:
        raise ProcessorError(f"failed to read Munki pkginfo {path}: {err}") from err


def repo_icon_path(munki_repo, pkginfo):
    icon_name = pkginfo.get("icon_name")
    if not icon_name:
        return None
    icon_path = os.path.join(munki_repo, "icons", str(icon_name))
    if os.path.isfile(icon_path):
        return icon_path
    if not os.path.splitext(icon_path)[1]:
        png_path = icon_path + ".png"
        if os.path.isfile(png_path):
            return png_path
    return None


def repo_package_path(munki_repo, location):
    if not location:
        return None
    package_path = os.path.join(munki_repo, "pkgs", str(location))
    if not os.path.isfile(package_path):
        raise ProcessorError(f"Munki package not found: {package_path}")
    return package_path


def import_counts():
    return {
        "pkginfos": 0,
        "software_created": 0,
        "packages_created": 0,
        "packages_updated": 0,
        "duplicate_pkginfos_skipped": 0,
        "package_binaries_uploaded": 0,
        "package_binaries_skipped": 0,
    }


def increment_action(counts, prefix, action):
    key = f"{prefix}_{action.lower()}"
    if key in counts:
        counts[key] += 1


def changed_count(counts):
    return (
        counts["software_created"]
        + counts["packages_created"]
        + counts["packages_updated"]
        + counts["package_binaries_uploaded"]
    )


if __name__ == "__main__":
    PROCESSOR = WoodstarMunkiRepoImporter()
    PROCESSOR.execute_shell()
