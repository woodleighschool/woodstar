#!/usr/local/autopkg/python

import os
import plistlib
import sys

from autopkglib import Processor, ProcessorError

sys.path.insert(0, os.path.dirname(__file__))

from WoodstarLib.Client import artifact_index, client_from_env, find_exact, truthy  # noqa: E402
from WoodstarMunkiAppUploader import WoodstarMunkiAppUploader  # noqa: E402
from WoodstarMunkiPackageUploader import WoodstarMunkiPackageUploader  # noqa: E402

__all__ = ["WoodstarMunkiRepoImporter"]


class WoodstarMunkiRepoImporter(Processor):
    description = "Imports a Munki repo's pkginfo items into Woodstar."

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
        artifacts = artifact_index(client)

        counts = import_counts()

        entries, skipped_duplicates = pkginfo_entries(munki_repo)
        counts["duplicate_pkginfos_skipped"] = skipped_duplicates
        if skipped_duplicates:
            self.output(f"Skipping {skipped_duplicates} duplicate Munki pkginfo item(s)")

        for pkginfo_path, pkginfo in entries:
            self.import_pkginfo(client, munki_repo, pkginfo_path, pkginfo, force, artifacts, counts)

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

    def munki_repo(self):
        munki_repo = self.env.get("MUNKI_REPO")
        if not munki_repo:
            raise ProcessorError("MUNKI_REPO is required")
        munki_repo = os.path.abspath(str(munki_repo))
        if not os.path.isdir(os.path.join(munki_repo, "pkgsinfo")):
            raise ProcessorError(f"MUNKI_REPO pkgsinfo directory was not found: {munki_repo}")
        return munki_repo

    def import_pkginfo(self, client, munki_repo, pkginfo_path, pkginfo, force, artifacts, counts):
        counts["pkginfos"] += 1
        app = WoodstarMunkiAppUploader()
        app.env = {
            "WOODSTAR_URL": self.env.get("WOODSTAR_URL"),
            "WOODSTAR_API_KEY": self.env.get("WOODSTAR_API_KEY"),
        }
        app.env["pkginfo"] = pkginfo

        name = app.software_name(pkginfo)
        if not name:
            raise ProcessorError(f"name is required for pkginfo {pkginfo_path}")
        self.output(f"Importing Munki pkginfo: {name} {pkginfo.get('version') or ''}")
        software, software_created = self.software_for_pkginfo(client, munki_repo, pkginfo, name, artifacts)
        if software_created:
            counts["software_created"] += 1

        package = WoodstarMunkiPackageUploader()
        package.env = {
            "WOODSTAR_URL": self.env.get("WOODSTAR_URL"),
            "WOODSTAR_API_KEY": self.env.get("WOODSTAR_API_KEY"),
            "MUNKI_REPO": munki_repo,
            "software_id": software["id"],
            "pkginfo_repo_path": pkginfo_path,
            "force": force,
        }
        installer_path = repo_package_path(munki_repo, pkginfo.get("installer_item_location"))
        if installer_path:
            package.env["pkg_repo_path"] = installer_path
        uninstaller_path = repo_package_path(munki_repo, pkginfo.get("uninstaller_item_location"))
        if uninstaller_path:
            package.env["uninstaller_pkg_path"] = uninstaller_path

        installer_path, uninstaller_path = package.artifact_paths(pkginfo)
        installer_artifact = None
        installer_uploaded = False
        if installer_path:
            installer_artifact, installer_uploaded = client.upload_artifact_status(
                "package",
                installer_path,
                os.path.basename(installer_path),
                force=force,
                artifact_index=artifacts,
            )
        uninstaller_artifact = None
        uninstaller_uploaded = False
        if uninstaller_path:
            uninstaller_artifact, uninstaller_uploaded = client.upload_artifact_status(
                "package",
                uninstaller_path,
                os.path.basename(uninstaller_path),
                force=force,
                artifact_index=artifacts,
            )

        body = package.package_body(pkginfo, int(software["id"]), installer_artifact, uninstaller_artifact)
        _package, package_action, package_changed = package.save_package(client, int(software["id"]), body)
        if package_changed:
            increment_action(counts, "packages", package_action)
        if installer_uploaded or uninstaller_uploaded:
            counts["package_binaries_uploaded"] += int(installer_uploaded) + int(uninstaller_uploaded)
        elif installer_artifact or uninstaller_artifact:
            counts["package_binaries_skipped"] += int(bool(installer_artifact)) + int(bool(uninstaller_artifact))

    def software_for_pkginfo(self, client, munki_repo, pkginfo, name, artifacts):
        existing = find_exact(client, "/api/munki/software", "name", name)
        if existing:
            return client.get(f"/api/munki/software/{existing['id']}"), False

        icon_artifact = None
        icon_path = repo_icon_path(munki_repo, pkginfo)
        if icon_path:
            icon_artifact, _uploaded = client.upload_artifact_status(
                "icon",
                icon_path,
                os.path.basename(icon_path),
                artifact_index=artifacts,
            )

        body = {
            "name": name,
            "description": pkginfo.get("description") or "",
            "category": pkginfo.get("category") or "",
            "developer": pkginfo.get("developer") or "",
            "targets": {"include": [], "exclude": []},
        }
        if icon_artifact:
            body["icon_artifact_id"] = icon_artifact["id"]
        self.output(f"Creating Woodstar Munki software: {name}")
        return client.post("/api/munki/software", body), True


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
    name = pkginfo.get("display_name") or pkginfo.get("name")
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
