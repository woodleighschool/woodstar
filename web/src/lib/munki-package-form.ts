import { z } from "zod";

import type { MunkiPackage, MunkiPackageMutation } from "@/hooks/munki/packages";
import type {
  PackageAlert,
  PackageInstallerEnvironmentVariable,
  PackageInstallItem,
  PackageItemToCopy,
  PackageReceipt,
  PackageReference,
} from "@/lib/api-client/types.gen";
import { fieldErrors, optionalText, requiredString } from "@/lib/form-validation";
import type { MunkiInstallerType, MunkiRestartAction, MunkiUninstallMethod } from "@/lib/munki-software-title";

export type Architecture = "arm64" | "x86_64";
export type ScriptKey =
  | "installcheck_script"
  | "uninstallcheck_script"
  | "preinstall_script"
  | "postinstall_script"
  | "preuninstall_script"
  | "postuninstall_script"
  | "uninstall_script"
  | "version_script";

export interface PackageReferenceRow extends PackageReference {
  rowID: string;
}

export interface StringRow {
  rowID: string;
  value: string;
}

export interface InstallerEnvironmentRow extends PackageInstallerEnvironmentVariable {
  rowID: string;
}

export interface InstallItemRow extends PackageInstallItem {
  rowID: string;
}

export interface ReceiptRow extends PackageReceipt {
  rowID: string;
}

export interface ItemToCopyRow extends PackageItemToCopy {
  rowID: string;
}

export interface PackageFormState {
  name: string;
  version: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
  installer_type: MunkiInstallerType;
  uninstall_method: MunkiUninstallMethod;
  custom_uninstall_method: string;
  restart_action: MunkiRestartAction;
  minimum_munki_version: string;
  minimum_os_version: string;
  maximum_os_version: string;
  supported_architectures: Architecture[];
  blocking_applications: StringRow[];
  requires: PackageReferenceRow[];
  update_for: PackageReferenceRow[];
  eligible: boolean;
  unattended_install: boolean;
  unattended_uninstall: boolean;
  uninstallable: boolean;
  on_demand: boolean;
  precache: boolean;
  autoremove: boolean;
  apple_item: boolean;
  suppress_bundle_relocation: boolean;
  force_install_after_date: string;
  installed_size: string;
  payload_identifier: string;
  package_path: string;
  installer_choices_xml: string;
  installer_environment: InstallerEnvironmentRow[];
  installs: InstallItemRow[];
  receipts: ReceiptRow[];
  items_to_copy: ItemToCopyRow[];
  notes: string;
  installcheck_script: string;
  uninstallcheck_script: string;
  preinstall_script: string;
  postinstall_script: string;
  preuninstall_script: string;
  postuninstall_script: string;
  uninstall_script: string;
  version_script: string;
  preinstall_alert: PackageAlert;
  preuninstall_alert: PackageAlert;
}

export const scriptFields: { key: ScriptKey; label: string }[] = [
  { key: "installcheck_script", label: "Install Check" },
  { key: "uninstallcheck_script", label: "Uninstall Check" },
  { key: "preinstall_script", label: "Preinstall" },
  { key: "postinstall_script", label: "Postinstall" },
  { key: "preuninstall_script", label: "Preuninstall" },
  { key: "postuninstall_script", label: "Postuninstall" },
  { key: "uninstall_script", label: "Uninstall" },
  { key: "version_script", label: "Version" },
];

export const packageIdentitySchema = z.object({
  name: requiredString("Name"),
  version: requiredString("Version"),
});

export function validatePackageForm({ value }: { value: PackageFormState }) {
  const result = packageIdentitySchema.safeParse(value);
  if (result.success) return undefined;
  return { fields: fieldErrors(result) };
}

export function packageMutationFromForm(
  form: PackageFormState,
  softwareID: number,
  artifacts: {
    installerArtifactID?: number;
    uninstallerArtifactID?: number;
    iconArtifactID?: number;
  },
): MunkiPackageMutation {
  return {
    software_id: softwareID,
    name: form.name,
    version: form.version,
    display_name: optionalText(form.display_name),
    description: optionalText(form.description),
    category: optionalText(form.category),
    developer: optionalText(form.developer),
    installer_type: form.installer_type,
    uninstall_method: form.uninstall_method,
    custom_uninstall_method: optionalText(form.custom_uninstall_method),
    restart_action: form.restart_action === "None" ? undefined : form.restart_action,
    minimum_munki_version: optionalText(form.minimum_munki_version),
    minimum_os_version: optionalText(form.minimum_os_version),
    maximum_os_version: optionalText(form.maximum_os_version),
    supported_architectures: form.supported_architectures,
    blocking_applications: cleanStringRows(form.blocking_applications),
    requires: cleanPackageReferences(form.requires),
    update_for: cleanPackageReferences(form.update_for),
    eligible: form.eligible,
    unattended_install: form.unattended_install,
    unattended_uninstall: form.unattended_uninstall,
    uninstallable: form.uninstallable,
    on_demand: form.on_demand,
    precache: form.precache,
    autoremove: form.autoremove,
    apple_item: form.apple_item,
    suppress_bundle_relocation: form.suppress_bundle_relocation,
    force_install_after_date: dateTimeLocalToISO(form.force_install_after_date),
    installed_size: numberOrUndefined(form.installed_size),
    payload_identifier: optionalText(form.payload_identifier),
    package_path: optionalText(form.package_path),
    installer_choices_xml: optionalText(form.installer_choices_xml),
    installer_environment: cleanInstallerEnvironment(form.installer_environment),
    installs: cleanInstallItems(form.installs),
    receipts: cleanReceipts(form.receipts),
    items_to_copy: cleanItemsToCopy(form.items_to_copy),
    notes: optionalText(form.notes),
    installcheck_script: optionalText(form.installcheck_script),
    uninstallcheck_script: optionalText(form.uninstallcheck_script),
    preinstall_script: optionalText(form.preinstall_script),
    postinstall_script: optionalText(form.postinstall_script),
    preuninstall_script: optionalText(form.preuninstall_script),
    postuninstall_script: optionalText(form.postuninstall_script),
    uninstall_script: optionalText(form.uninstall_script),
    version_script: optionalText(form.version_script),
    preinstall_alert: cleanAlert(form.preinstall_alert),
    preuninstall_alert: cleanAlert(form.preuninstall_alert),
    installer_artifact_id: artifacts.installerArtifactID,
    uninstaller_artifact_id: artifacts.uninstallerArtifactID,
    icon_artifact_id: artifacts.iconArtifactID,
  };
}

export function emptyPackageForm(): PackageFormState {
  return {
    name: "",
    version: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
    installer_type: "pkg",
    uninstall_method: "none",
    custom_uninstall_method: "",
    restart_action: "None",
    minimum_munki_version: "",
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
    blocking_applications: [],
    requires: [],
    update_for: [],
    eligible: true,
    unattended_install: true,
    unattended_uninstall: true,
    uninstallable: false,
    on_demand: false,
    precache: false,
    autoremove: false,
    apple_item: false,
    suppress_bundle_relocation: false,
    force_install_after_date: "",
    installed_size: "",
    payload_identifier: "",
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

export function packageFormFromPackage(pkg: MunkiPackage): PackageFormState {
  return {
    name: pkg.name,
    version: pkg.version,
    display_name: pkg.display_name,
    description: pkg.description,
    category: pkg.category,
    developer: pkg.developer,
    installer_type: pkg.installer_type,
    uninstall_method: pkg.uninstall_method,
    custom_uninstall_method: pkg.custom_uninstall_method,
    restart_action: pkg.restart_action ?? "None",
    minimum_munki_version: pkg.minimum_munki_version,
    minimum_os_version: pkg.minimum_os_version,
    maximum_os_version: pkg.maximum_os_version,
    supported_architectures: (pkg.supported_architectures ?? []).filter(isArchitecture),
    blocking_applications: stringRows(pkg.blocking_applications ?? []),
    requires: packageReferenceRows(pkg.requires ?? []),
    update_for: packageReferenceRows(pkg.update_for ?? []),
    eligible: pkg.eligible,
    unattended_install: pkg.unattended_install,
    unattended_uninstall: pkg.unattended_uninstall,
    uninstallable: pkg.uninstallable,
    on_demand: pkg.on_demand,
    precache: pkg.precache,
    autoremove: pkg.autoremove,
    apple_item: pkg.apple_item,
    suppress_bundle_relocation: pkg.suppress_bundle_relocation,
    force_install_after_date: isoToDateTimeLocal(pkg.force_install_after_date),
    installed_size: pkg.installed_size > 0 ? String(pkg.installed_size) : "",
    payload_identifier: pkg.payload_identifier,
    package_path: pkg.package_path,
    installer_choices_xml: pkg.installer_choices_xml,
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

function emptyAlert(): PackageAlert {
  return { enabled: false };
}

export function emptyStringRow(): StringRow {
  return { rowID: rowID(), value: "" };
}

export function emptyPackageReferenceRow(): PackageReferenceRow {
  return { rowID: rowID(), name: "" };
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

function packageReferenceRows(values: PackageReference[]): PackageReferenceRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function stringRows(values: string[]): StringRow[] {
  return values.map((value) => ({ rowID: rowID(), value }));
}

function installerEnvironmentRows(values: PackageInstallerEnvironmentVariable[]): InstallerEnvironmentRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function installItemRows(values: PackageInstallItem[]): InstallItemRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function receiptRows(values: PackageReceipt[]): ReceiptRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function itemToCopyRows(values: PackageItemToCopy[]): ItemToCopyRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function cleanPackageReferences(rows: PackageReferenceRow[]): PackageReference[] {
  const out: PackageReference[] = [];
  for (const row of rows) {
    if (row.package_id) {
      out.push({ package_id: row.package_id });
      continue;
    }
    const name = row.name?.trim();
    if (name) out.push({ name });
  }
  return out;
}

function cleanInstallerEnvironment(rows: InstallerEnvironmentRow[]): PackageInstallerEnvironmentVariable[] {
  return rows.flatMap((row) => {
    const name = row.name.trim();
    return name ? [{ name, value: row.value }] : [];
  });
}

function cleanInstallItems(rows: InstallItemRow[]): PackageInstallItem[] {
  return rows.flatMap((row) => {
    const path = row.path.trim();
    return path ? [{ ...stripRowID(row), path }] : [];
  });
}

function cleanReceipts(rows: ReceiptRow[]): PackageReceipt[] {
  return rows.flatMap((row) => {
    const packageID = row.package_id.trim();
    return packageID
      ? [{ package_id: packageID, version: optionalText(row.version ?? ""), optional: row.optional }]
      : [];
  });
}

function cleanItemsToCopy(rows: ItemToCopyRow[]): PackageItemToCopy[] {
  return rows.flatMap((row) => {
    const sourceItem = row.source_item.trim();
    const destinationPath = row.destination_path.trim();
    return sourceItem || destinationPath
      ? [{ ...stripRowID(row), source_item: sourceItem, destination_path: destinationPath }]
      : [];
  });
}

function cleanAlert(alert: PackageAlert): PackageAlert {
  if (!alert.enabled) return { enabled: false };
  return {
    enabled: true,
    title: optionalText(alert.title ?? ""),
    detail: optionalText(alert.detail ?? ""),
    ok_label: optionalText(alert.ok_label ?? ""),
    cancel_label: optionalText(alert.cancel_label ?? ""),
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

export function replaceAt<T>(rows: T[], index: number, row: T) {
  return rows.map((value, rowIndex) => (rowIndex === index ? row : value));
}

export function removeAt<T>(rows: T[], index: number) {
  return rows.filter((_, rowIndex) => rowIndex !== index);
}

export function toggleArray<T>(values: T[], value: T, enabled: boolean) {
  if (enabled) return Array.from(new Set([...values, value]));
  return values.filter((item) => item !== value);
}

export function packageLabel(pkg: MunkiPackage) {
  return `${pkg.name} ${pkg.version}`;
}

export function packageReferenceLabel(row: PackageReferenceRow, packages: MunkiPackage[]) {
  const pkg = packages.find((item) => item.id === row.package_id);
  if (pkg) return packageLabel(pkg);
  if (row.package_name) return `${row.package_name} ${row.package_version ?? ""}`.trim();
  return "";
}

function isArchitecture(value: string): value is Architecture {
  return value === "arm64" || value === "x86_64";
}

function rowID() {
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
