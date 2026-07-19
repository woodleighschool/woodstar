import type {
  MunkiPackage,
  MunkiPackageAlert,
  MunkiPackageInstallerChoice,
  MunkiPackageInstallerEnvironmentVariable,
  MunkiPackageInstallItem,
  MunkiPackageItemToCopy,
  MunkiPackageMutation,
  MunkiPackageReceipt,
  MunkiPackageReference,
  MunkiPackageReferenceMutation,
} from "@/lib/api";
import { nonEmpty } from "@/lib/utils";

import {
  type InstallItemRow,
  type InstallerEnvironmentRow,
  type ItemToCopyRow,
  type PackageFormInput,
  type PackageFormOutput,
  type PackageReferenceRow,
  type ReceiptRow,
  type StringRow,
  isArchitecture,
  parseInstallerChoices,
} from "./form-schema";

export type PackageFormMutation = MunkiPackageMutation & {
  installer_type: PackageFormOutput["installer_type"];
};

export function packageMutationFromForm(
  form: PackageFormOutput,
  installerObjectID = form.installer_object_id ?? undefined,
): PackageFormMutation {
  const installerType = form.installer_type;
  const uninstallMethod = form.uninstall_method;
  const usesInstallerOptions = installerType !== "nopkg";
  const usesItemsToCopy =
    installerType === "copy_from_dmg" || uninstallMethod === "remove_copied_items";
  const blockingApplications = cleanStringRows(form.blocking_applications);

  const mutation = {
    version: form.version,
    installer_type: installerType,
    installer_object_id: installerType === "nopkg" ? undefined : installerObjectID,
    uninstallable: form.uninstallable,
    uninstall_method: uninstallMethod || undefined,
    restart_action: form.restart_required ? form.restart_action : undefined,
    minimum_munki_version: nonEmpty(form.minimum_munki_version),
    minimum_os_version: nonEmpty(form.minimum_os_version),
    maximum_os_version: nonEmpty(form.maximum_os_version),
    supported_architectures: form.supported_architectures,
    blocking_applications_none: form.blocking_applications_none,
    blocking_applications: form.blocking_applications_none ? [] : blockingApplications,
    installable_condition: nonEmpty(form.installable_condition),
    blocking_applications_manual_quit_only: form.blocking_applications_manual_quit_only,
    blocking_applications_quit_script: nonEmpty(form.blocking_applications_quit_script),
    requires: cleanPackageReferences(form.requires),
    update_for: cleanPackageReferences(form.update_for),
    unattended_install: form.unattended_install,
    unattended_uninstall: form.unattended_uninstall,
    on_demand: form.on_demand,
    precache: form.precache,
    autoremove: form.autoremove,
    apple_item: form.apple_item,
    suppress_bundle_relocation: form.suppress_bundle_relocation,
    force_install_after_date: dateTimeLocalToISO(form.force_install_after_date),
    installed_size: numberOrUndefined(form.installed_size),
    package_path: usesInstallerOptions ? nonEmpty(form.package_path) : undefined,
    installer_choices_xml: usesInstallerOptions
      ? parseInstallerChoices(form.installer_choices_xml)
      : [],
    installer_environment: usesInstallerOptions
      ? cleanInstallerEnvironment(form.installer_environment)
      : [],
    installs: cleanInstallItems(form.installs),
    receipts: cleanReceipts(form.receipts),
    items_to_copy: usesItemsToCopy ? cleanItemsToCopy(form.items_to_copy) : [],
    notes: nonEmpty(form.notes),
    installcheck_script: nonEmpty(form.installcheck_script),
    uninstallcheck_script: nonEmpty(form.uninstallcheck_script),
    preinstall_script: nonEmpty(form.preinstall_script),
    postinstall_script: nonEmpty(form.postinstall_script),
    preuninstall_script: nonEmpty(form.preuninstall_script),
    postuninstall_script: nonEmpty(form.postuninstall_script),
    uninstall_script:
      uninstallMethod === "uninstall_script" ? nonEmpty(form.uninstall_script) : undefined,
    version_script: nonEmpty(form.version_script),
    preinstall_alert: cleanAlert(form.preinstall_alert),
    preuninstall_alert: cleanAlert(form.preuninstall_alert),
  } satisfies Record<keyof MunkiPackageMutation, unknown>;

  return mutation;
}

export function emptyPackageForm(softwareID: number | null = null): PackageFormInput {
  return {
    software_id: softwareID,
    version: "",
    installer_type: "pkg",
    installer_object_id: null,
    installer_file: null,
    uninstallable: false,
    uninstall_method: "",
    restart_required: false,
    restart_action: "RequireRestart",
    minimum_munki_version: "",
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
    blocking_applications_none: false,
    blocking_applications: [],
    installable_condition: "",
    blocking_applications_manual_quit_only: false,
    blocking_applications_quit_script: "",
    requires: [],
    update_for: [],
    unattended_install: false,
    unattended_uninstall: false,
    on_demand: false,
    precache: false,
    autoremove: false,
    apple_item: false,
    suppress_bundle_relocation: false,
    force_install_after_date: "",
    installed_size: "",
    package_path: "",
    installer_choices_xml: "",
    installer_environment: [],
    installs: [],
    receipts: [],
    items_to_copy: [],
    notes: "",
    installcheck_script: "",
    uninstallcheck_script: "",
    preinstall_script: "",
    postinstall_script: "",
    preuninstall_script: "",
    postuninstall_script: "",
    uninstall_script: "",
    version_script: "",
    preinstall_alert: emptyAlert(),
    preuninstall_alert: emptyAlert(),
  };
}

export function packageFormFromPackage(pkg: MunkiPackage): PackageFormInput {
  return {
    software_id: pkg.software.id,
    version: pkg.version,
    installer_type: pkg.installer_type,
    installer_object_id: pkg.installer_object_id ?? null,
    installer_file: null,
    uninstallable: pkg.uninstallable,
    uninstall_method: pkg.uninstall_method ?? "",
    restart_required: pkg.restart_action !== undefined,
    restart_action: pkg.restart_action ?? "RequireRestart",
    minimum_munki_version: pkg.minimum_munki_version,
    minimum_os_version: pkg.minimum_os_version,
    maximum_os_version: pkg.maximum_os_version,
    supported_architectures: (pkg.supported_architectures ?? []).filter(isArchitecture),
    blocking_applications_none: pkg.blocking_applications_none,
    blocking_applications: pkg.blocking_applications_none
      ? []
      : stringRows(pkg.blocking_applications ?? []),
    installable_condition: pkg.installable_condition,
    blocking_applications_manual_quit_only: pkg.blocking_applications_manual_quit_only,
    blocking_applications_quit_script: pkg.blocking_applications_quit_script,
    requires: packageReferenceRows(pkg.requires ?? []),
    update_for: packageReferenceRows(pkg.update_for ?? []),
    unattended_install: pkg.unattended_install,
    unattended_uninstall: pkg.unattended_uninstall,
    on_demand: pkg.on_demand,
    precache: pkg.precache,
    autoremove: pkg.autoremove,
    apple_item: pkg.apple_item,
    suppress_bundle_relocation: pkg.suppress_bundle_relocation,
    force_install_after_date: isoToDateTimeLocal(pkg.force_install_after_date),
    installed_size: pkg.installed_size > 0 ? String(pkg.installed_size) : "",
    package_path: pkg.package_path,
    installer_choices_xml: installerChoicesText(pkg.installer_choices_xml ?? []),
    installer_environment: installerEnvironmentRows(pkg.installer_environment ?? []),
    installs: installItemRows(pkg.installs ?? []),
    receipts: receiptRows(pkg.receipts ?? []),
    items_to_copy: itemToCopyRows(pkg.items_to_copy ?? []),
    notes: pkg.notes,
    installcheck_script: pkg.installcheck_script,
    uninstallcheck_script: pkg.uninstallcheck_script,
    preinstall_script: pkg.preinstall_script,
    postinstall_script: pkg.postinstall_script,
    preuninstall_script: pkg.preuninstall_script,
    postuninstall_script: pkg.postuninstall_script,
    uninstall_script: pkg.uninstall_script,
    version_script: pkg.version_script,
    preinstall_alert: pkg.preinstall_alert,
    preuninstall_alert: pkg.preuninstall_alert,
  };
}

export function emptyStringRow(): StringRow {
  return { rowID: rowID(), value: "" };
}

export function emptyPackageReferenceRow(): PackageReferenceRow {
  return { rowID: rowID() };
}

export function emptyInstallerEnvironmentRow(): InstallerEnvironmentRow {
  return { rowID: rowID(), name: "", value: "" };
}

export function emptyInstallItemRow(): InstallItemRow {
  return { rowID: rowID(), type: "application", path: "" };
}

export function emptyReceiptRow(): ReceiptRow {
  return { rowID: rowID(), package_id: "" };
}

export function emptyItemToCopyRow(): ItemToCopyRow {
  return { rowID: rowID(), source_item: "", destination_path: "" };
}

function emptyAlert(): MunkiPackageAlert {
  return { enabled: false };
}

function packageReferenceRows(values: MunkiPackageReference[]): PackageReferenceRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function stringRows(values: string[]): StringRow[] {
  return values.map((value) => ({ rowID: rowID(), value }));
}

function installerEnvironmentRows(
  values: MunkiPackageInstallerEnvironmentVariable[],
): InstallerEnvironmentRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function installItemRows(values: MunkiPackageInstallItem[]): InstallItemRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function receiptRows(values: MunkiPackageReceipt[]): ReceiptRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function itemToCopyRows(values: MunkiPackageItemToCopy[]): ItemToCopyRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function cleanPackageReferences(rows: PackageReferenceRow[]): MunkiPackageReferenceMutation[] {
  const out: MunkiPackageReferenceMutation[] = [];
  for (const row of rows) {
    if (row.software_id) {
      out.push({
        software_id: row.software_id,
        package_id: row.package_id,
      });
    }
  }
  return out;
}

function cleanInstallerEnvironment(
  rows: InstallerEnvironmentRow[],
): MunkiPackageInstallerEnvironmentVariable[] {
  return rows.flatMap((row) => {
    const name = row.name.trim();
    return name ? [{ name, value: row.value }] : [];
  });
}

function installerChoicesText(values: MunkiPackageInstallerChoice[]) {
  return values.length === 0 ? "" : JSON.stringify(values, null, 2);
}

function cleanInstallItems(rows: InstallItemRow[]): MunkiPackageInstallItem[] {
  return rows.flatMap((row) => {
    const path = row.path.trim();
    return path ? [{ ...stripRowID(row), path }] : [];
  });
}

function cleanReceipts(rows: ReceiptRow[]): MunkiPackageReceipt[] {
  return rows.flatMap((row) => {
    const packageID = row.package_id.trim();
    return packageID
      ? [
          {
            package_id: packageID,
            version: nonEmpty(row.version ?? ""),
            name: nonEmpty(row.name ?? ""),
            installed_size: row.installed_size,
            optional: row.optional,
          },
        ]
      : [];
  });
}

function cleanItemsToCopy(rows: ItemToCopyRow[]): MunkiPackageItemToCopy[] {
  return rows.flatMap((row) => {
    const sourceItem = row.source_item.trim();
    const destinationPath = row.destination_path.trim();
    return sourceItem || destinationPath
      ? [{ ...stripRowID(row), source_item: sourceItem, destination_path: destinationPath }]
      : [];
  });
}

function cleanAlert(alert: MunkiPackageAlert): MunkiPackageAlert {
  if (!alert.enabled) return { enabled: false };
  return {
    enabled: true,
    title: nonEmpty(alert.title ?? ""),
    detail: nonEmpty(alert.detail ?? ""),
    ok_label: nonEmpty(alert.ok_label ?? ""),
    cancel_label: nonEmpty(alert.cancel_label ?? ""),
  };
}

function cleanStringRows(values: StringRow[]) {
  return values.map((row) => row.value.trim()).filter(Boolean);
}

function numberOrUndefined(value: string) {
  if (value.trim() === "") return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function dateTimeLocalToISO(value: string) {
  if (value.trim() === "") return undefined;
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? undefined : date.toISOString();
}

function isoToDateTimeLocal(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return "";
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.valueOf() - offset).toISOString().slice(0, 16);
}

function stripRowID<T extends { rowID: string }>(row: T): Omit<T, "rowID"> {
  const { rowID: _rowID, ...rest } = row;
  return rest;
}

function rowID() {
  return crypto.randomUUID();
}
