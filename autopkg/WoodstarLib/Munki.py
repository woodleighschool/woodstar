import os.path

from autopkglib import ProcessorError

from WoodstarLib.Client import find_exact, list_items, needs_object

UNSET = object()

SUPPORTED_TARGET_ACTIONS = {
    "managed_installs",
    "managed_uninstalls",
    "managed_updates",
    "optional_installs",
    "featured_items",
    "default_installs",
}

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
    "uninstallable": False,
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


class SoftwareManager:
    def __init__(self, client, output=None):
        self.client = client
        self.output = output or (lambda _message: None)

    @staticmethod
    def software_name(pkginfo, name=UNSET):
        value = pkginfo.get("name") if name is UNSET else name
        if not isinstance(value, str) or not value.strip():
            raise ProcessorError("name is required")
        return value.strip()

    @staticmethod
    def software_metadata(pkginfo, name, overrides=None):
        overrides = overrides or {}
        body = {}
        for key in ("display_name", "description", "category", "developer"):
            value = overrides[key] if key in overrides else pkginfo.get(key, "")
            if not isinstance(value, str):
                raise ProcessorError(f"{key} must be a string")
            body[key] = value
        if body["display_name"].strip() == name:
            body["display_name"] = ""
        return body

    def prepare(self, pkginfo, name=UNSET, metadata=None, targets=UNSET):
        resolved_name = self.software_name(pkginfo, name)
        existing = find_exact(
            self.client,
            "/api/munki/software",
            "name",
            resolved_name,
        )
        if existing:
            existing = self.client.get(f"/api/munki/software/{existing['id']}")

        body = self.software_metadata(pkginfo, resolved_name, metadata)
        if existing and existing.get("icon_object_id") is not None:
            body["icon_object_id"] = existing["icon_object_id"]
        body["targets"] = self.target_body(targets, existing)
        return resolved_name, existing, body

    def save(self, name, existing, body):
        if existing and self.software_matches(existing, body):
            return existing, "Skipped", False
        if existing:
            self.output(f"Updating Woodstar Munki software: {name}")
            saved = self.client.put(f"/api/munki/software/{existing['id']}", body)
            return saved, "Updated", True
        self.output(f"Creating Woodstar Munki software: {name}")
        saved = self.client.post("/api/munki/software", {"name": name, **body})
        return saved, "Created", True

    def attach_icon(self, software, icon_path, force=False):
        if not icon_path or not needs_object(software, "icon", icon_path, force):
            return software, False
        self.client.attach_object(
            f"/api/munki/software/{software['id']}/icon",
            icon_path,
        )
        return self.client.get(f"/api/munki/software/{software['id']}"), True

    @staticmethod
    def software_matches(existing, body):
        return all(
            normalized(existing.get(key)) == normalized(value)
            for key, value in body.items()
        )

    def target_body(self, target_config=UNSET, existing=None):
        if target_config is UNSET:
            return existing["targets"] if existing else empty_targets()
        if not isinstance(target_config, dict):
            raise ProcessorError("targets must be a dictionary")
        include_inputs = target_config.get("include", [])
        if not isinstance(include_inputs, list):
            raise ProcessorError("targets.include must be a list")
        exclude_inputs = target_config.get("exclude", [])
        if not isinstance(exclude_inputs, list):
            raise ProcessorError("targets.exclude must be a list")
        return {
            "include": [self.include_body(target) for target in include_inputs],
            "exclude": self.exclude_body(exclude_inputs),
        }

    def include_body(self, target):
        if not isinstance(target, dict):
            raise ProcessorError("targets.include entries must be dictionaries")
        if "priority" in target:
            raise ProcessorError(
                "targets.include priority is not supported; order the include list instead"
            )
        return {
            "label_id": int(self.label_id(target, "targets.include entry")),
            "package": self.package_selector(target),
            "actions": self.actions(target),
        }

    @staticmethod
    def actions(target):
        actions = target.get("actions", ["managed_installs"])
        if not isinstance(actions, list) or not actions:
            raise ProcessorError("targets.include actions must be a non-empty list")
        for action in actions:
            if action not in SUPPORTED_TARGET_ACTIONS:
                raise ProcessorError(
                    f"targets.include action {action!r} is not supported"
                )
        return actions

    @staticmethod
    def package_selector(target):
        package = target.get("package", {"strategy": "latest"})
        if not isinstance(package, dict):
            raise ProcessorError("targets.include package must be a dictionary")
        strategy = package.get("strategy", "latest")
        if strategy not in {"latest", "specific"}:
            raise ProcessorError(
                "targets.include package.strategy must be latest or specific"
            )
        if strategy == "latest":
            if package.get("package_id"):
                raise ProcessorError(
                    "targets.include latest package must not set package_id"
                )
            return {"strategy": "latest"}
        package_id = package.get("package_id")
        if not package_id:
            raise ProcessorError(
                "targets.include specific package requires package_id"
            )
        return {"strategy": "specific", "package_id": int(package_id)}

    def exclude_body(self, inputs):
        values = [
            self.label_ref(target, "targets.exclude entry") for target in inputs
        ]
        label_ids = [target["label_id"] for target in values]
        if len(label_ids) != len(set(label_ids)):
            raise ProcessorError("targets.exclude contains duplicate labels")
        return values

    def label_ref(self, target, context):
        if isinstance(target, (int, str)):
            return {"label_id": int(target)}
        if not isinstance(target, dict):
            raise ProcessorError(f"{context} must be a dictionary")
        return {"label_id": int(self.label_id(target, context))}

    def label_id(self, target, context):
        label_id = target.get("label_id")
        if label_id:
            return label_id
        label_name = target.get("label_name")
        if not label_name:
            raise ProcessorError(f"{context} requires label_id or label_name")
        label = find_exact(self.client, "/api/labels", "name", label_name)
        if not label:
            raise ProcessorError(f"Woodstar label not found: {label_name}")
        return label["id"]


class PackageManager:
    def __init__(self, client, output=None):
        self.client = client
        self.output = output or (lambda _message: None)

    def installer_artifact_path(self, pkginfo, pkg_path):
        if not self.needs_installer_artifact(pkginfo):
            return None
        if not pkg_path or not os.path.isfile(pkg_path):
            raise ProcessorError(
                "package-bearing pkginfo requires an installer artifact"
            )
        return os.path.abspath(pkg_path)

    @classmethod
    def needs_installer_artifact(cls, pkginfo):
        return cls.installer_type(pkginfo) != "nopkg"

    def base_package_body(self, pkginfo, software_id):
        body = self.package_mutation_from_pkginfo(pkginfo)
        body["software_id"] = software_id
        return body

    def package_body(self, pkginfo, software_id, resolver):
        body = self.base_package_body(pkginfo, software_id)
        body["requires"] = resolver.resolve(pkginfo.get("requires", []), "requires")
        body["update_for"] = resolver.resolve(
            pkginfo.get("update_for", []),
            "update_for",
        )
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
        body["uninstallable"] = self.uninstallable(pkginfo)
        uninstall_method = self.uninstall_method(pkginfo)
        if uninstall_method:
            body["uninstall_method"] = uninstall_method
        for key in PACKAGE_DIRECT_KEYS:
            if key in pkginfo:
                body[key] = pkginfo[key]
        if "blocking_applications" in pkginfo:
            body["blocking_applications_none"] = (
                len(pkginfo["blocking_applications"]) == 0
            )
        force_date = body.get("force_install_after_date")
        if hasattr(force_date, "isoformat"):
            body["force_install_after_date"] = force_date.isoformat()
        if "RestartAction" in pkginfo:
            body["restart_action"] = pkginfo["RestartAction"]
        if "OnDemand" in pkginfo:
            body["on_demand"] = pkginfo["OnDemand"]
        if "installer_environment" in pkginfo:
            body["installer_environment"] = self.installer_environment(
                pkginfo["installer_environment"]
            )
        if "installer_choices_xml" in pkginfo:
            body["installer_choices_xml"] = self.installer_choices_xml(
                pkginfo["installer_choices_xml"]
            )
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

    def save_package(self, software_id, body, installer_path, force=False):
        existing = self.existing_package(software_id, body["version"])
        if existing:
            existing = self.client.get(f"/api/munki/packages/{existing['id']}")
        installer_object_id = None
        installer_uploaded = False
        if body["installer_type"] != "nopkg":
            if not installer_path:
                raise ProcessorError(
                    "package-bearing pkginfo requires an installer artifact"
                )
            if not existing or needs_object(
                existing,
                "installer",
                installer_path,
                force,
            ):
                installer = self.client.upload_package_installer(installer_path)
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
                saved = self.client.put(
                    f"/api/munki/packages/{existing['id']}",
                    update_body,
                )
                return saved, "Updated", True, installer_uploaded
            saved = self.client.post(
                "/api/munki/packages",
                mutation_request_body(body),
            )
            return saved, "Created", True, installer_uploaded
        except ProcessorError:
            if installer_uploaded:
                try:
                    self.client.delete(
                        f"/api/munki/package-installers/{installer_object_id}"
                    )
                except ProcessorError:
                    pass
            raise

    def existing_package(self, software_id, version):
        for item in list_items(
            self.client,
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
    def uninstallable(pkginfo):
        value = pkginfo.get("uninstallable", False)
        if not isinstance(value, bool):
            raise ProcessorError("pkginfo uninstallable must be a boolean")
        return value

    @staticmethod
    def installer_environment(value):
        if not isinstance(value, dict):
            raise ProcessorError(
                "pkginfo installer_environment must be a dictionary"
            )
        return [{"name": name, "value": value[name]} for name in sorted(value)]

    @staticmethod
    def installer_choices_xml(values):
        if not isinstance(values, list):
            raise ProcessorError("pkginfo installer_choices_xml must be a list")
        choices = []
        for value in values:
            if not isinstance(value, dict):
                raise ProcessorError(
                    "pkginfo installer_choices_xml entries must be dictionaries"
                )
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
                raise ProcessorError(
                    "pkginfo installer_choices_xml attributeSetting must be an integer"
                ) from err
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


def empty_targets():
    return {"include": [], "exclude": []}


def default_package_mutation():
    return {
        key: (
            [*value]
            if isinstance(value, list)
            else dict(value)
            if isinstance(value, dict)
            else value
        )
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
        return {
            key: normalized(item)
            for key, item in sorted(value.items())
            if item is not None
        }
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
            raise ProcessorError(
                f"pkginfo {key} item {value!r} was not found in Woodstar"
            )
        unique = {tuple(sorted(candidate.items())) for candidate in candidates}
        if len(unique) != 1:
            raise ProcessorError(
                f"pkginfo {key} item {value!r} is ambiguous in Woodstar"
            )
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
