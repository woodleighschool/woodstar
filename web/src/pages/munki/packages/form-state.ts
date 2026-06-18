import { z } from "zod";

import type { MunkiPackage, MunkiPackageMutation } from "@/hooks/use-munki-packages";
import type {
  MunkiPackageAlert,
  MunkiPackageInstallerChoice,
  MunkiPackageInstallerEnvironmentVariable,
  MunkiPackageInstallItem,
  MunkiPackageItemToCopy,
  MunkiPackageReceipt,
  MunkiPackageReference,
} from "@/lib/api";
import { fieldErrors, requiredString } from "@/lib/form-validation";
import { nonEmpty } from "@/lib/utils";
import type {
  MunkiInstallerType,
  MunkiRestartAction,
  MunkiUninstallMethod,
} from "@/pages/munki/software/munki-software";

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

export interface PackageReferenceRow extends Omit<
  MunkiPackageReference,
  "package_id" | "software_id"
> {
  software_id?: number;
  package_id?: number;
  rowID: string;
}

export interface StringRow {
  rowID: string;
  value: string;
}

export interface InstallerEnvironmentRow extends MunkiPackageInstallerEnvironmentVariable {
  rowID: string;
}

export interface InstallItemRow extends MunkiPackageInstallItem {
  rowID: string;
}

export interface ReceiptRow extends MunkiPackageReceipt {
  rowID: string;
}

export interface ItemToCopyRow extends MunkiPackageItemToCopy {
  rowID: string;
}

export interface PackageFormState {
  version: string;
  installer_type: MunkiInstallerType;
  uninstall_method: MunkiUninstallMethod;
  restart_action: MunkiRestartAction;
  minimum_munki_version: string;
  minimum_os_version: string;
  maximum_os_version: string;
  supported_architectures: Architecture[];
  include_empty_blocking_applications: boolean;
  blocking_applications: StringRow[];
  installable_condition: string;
  blocking_applications_manual_quit_only: boolean;
  blocking_applications_quit_script: string;
  requires: PackageReferenceRow[];
  update_for: PackageReferenceRow[];
  eligible: boolean;
  unattended_install: boolean;
  unattended_uninstall: boolean;
  on_demand: boolean;
  precache: boolean;
  autoremove: boolean;
  apple_item: boolean;
  suppress_bundle_relocation: boolean;
  force_install_after_date: string;
  installed_size: string;
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
  preinstall_alert: MunkiPackageAlert;
  preuninstall_alert: MunkiPackageAlert;
}

export const scriptFields: { key: ScriptKey; label: string }[] = [
  { key: "preinstall_script", label: "Pre-install" },
  { key: "postinstall_script", label: "Post-install" },
  { key: "uninstall_script", label: "Uninstall" },
  { key: "preuninstall_script", label: "Pre-uninstall" },
  { key: "postuninstall_script", label: "Post-uninstall" },
  { key: "version_script", label: "Version" },
  { key: "installcheck_script", label: "Install Check" },
  { key: "uninstallcheck_script", label: "Uninstall Check" },
];

const packageIdentitySchema = z.object({
  version: requiredString("Version"),
});

export function validatePackageForm({ value }: { value: PackageFormState }) {
  const result = packageIdentitySchema.safeParse(value);
  if (result.success) return undefined;
  return { fields: fieldErrors(result) };
}

export function packageSubmitPreflightError(
  form: PackageFormState,
  files: { hasInstallerFile: boolean },
) {
  if (form.installer_type !== "nopkg" && !files.hasInstallerFile) {
    return `${installerTypeLabel(form.installer_type)} packages require an installer file.`;
  }
  if (
    form.installer_type === "copy_from_dmg" &&
    cleanItemsToCopy(form.items_to_copy).length === 0
  ) {
    return "Copy from DMG packages require at least one item to copy.";
  }
  if (form.installer_type !== "nopkg") {
    const installerChoicesError = parseInstallerChoicesError(form.installer_choices_xml);
    if (installerChoicesError) return installerChoicesError;
  }
  if (form.installer_type === "nopkg" && !hasNoPkgEvidence(form)) {
    return "No package items require an install check script, installs, receipts, or On Demand.";
  }
  if (form.uninstall_method === "removepackages" && cleanReceipts(form.receipts).length === 0) {
    return "Remove packages uninstall requires at least one receipt.";
  }
  if (
    form.uninstall_method === "remove_copied_items" &&
    cleanItemsToCopy(form.items_to_copy).length === 0
  ) {
    return "Remove copied items uninstall requires at least one item to copy.";
  }
  if (
    form.uninstall_method === "uninstall_script" &&
    nonEmpty(form.uninstall_script) === undefined
  ) {
    return "Uninstall script method requires an uninstall script.";
  }
  return undefined;
}

export function packageMutationFromForm(form: PackageFormState): MunkiPackageMutation {
  const installerType = form.installer_type;
  const uninstallMethod = form.uninstall_method;
  const usesInstallerOptions = installerType !== "nopkg";
  const usesItemsToCopy =
    installerType === "copy_from_dmg" || uninstallMethod === "remove_copied_items";
  const blockingApplications = cleanStringRows(form.blocking_applications);

  return {
    version: form.version,
    installer_type: installerType,
    uninstall_method: uninstallMethod,
    restart_action: form.restart_action === "None" ? undefined : form.restart_action,
    minimum_munki_version: nonEmpty(form.minimum_munki_version),
    minimum_os_version: nonEmpty(form.minimum_os_version),
    maximum_os_version: nonEmpty(form.maximum_os_version),
    supported_architectures: form.supported_architectures,
    blocking_applications:
      blockingApplications.length > 0
        ? blockingApplications
        : form.include_empty_blocking_applications
          ? []
          : undefined,
    installable_condition: nonEmpty(form.installable_condition),
    blocking_applications_manual_quit_only: form.blocking_applications_manual_quit_only,
    blocking_applications_quit_script: nonEmpty(form.blocking_applications_quit_script),
    requires: cleanPackageReferences(form.requires),
    update_for: cleanPackageReferences(form.update_for),
    eligible: form.eligible,
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
  };
}

export function emptyPackageForm(): PackageFormState {
  return {
    version: "",
    installer_type: "pkg",
    uninstall_method: "none",
    restart_action: "None",
    minimum_munki_version: "",
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
    include_empty_blocking_applications: false,
    blocking_applications: [],
    installable_condition: "",
    blocking_applications_manual_quit_only: false,
    blocking_applications_quit_script: "",
    requires: [],
    update_for: [],
    eligible: true,
    unattended_install: true,
    unattended_uninstall: true,
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

export function packageFormFromPackage(pkg: MunkiPackage): PackageFormState {
  return {
    version: pkg.version,
    installer_type: pkg.installer_type,
    uninstall_method: pkg.uninstall_method,
    restart_action: pkg.restart_action ?? "None",
    minimum_munki_version: pkg.minimum_munki_version,
    minimum_os_version: pkg.minimum_os_version,
    maximum_os_version: pkg.maximum_os_version,
    supported_architectures: (pkg.supported_architectures ?? []).filter(isArchitecture),
    include_empty_blocking_applications: pkg.blocking_applications?.length === 0,
    blocking_applications: stringRows(pkg.blocking_applications ?? []),
    installable_condition: pkg.installable_condition,
    blocking_applications_manual_quit_only: pkg.blocking_applications_manual_quit_only,
    blocking_applications_quit_script: pkg.blocking_applications_quit_script,
    requires: packageReferenceRows(pkg.requires ?? []),
    update_for: packageReferenceRows(pkg.update_for ?? []),
    eligible: pkg.eligible,
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

function emptyAlert(): MunkiPackageAlert {
  return { enabled: false };
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

function cleanPackageReferences(rows: PackageReferenceRow[]): MunkiPackageReference[] {
  const out: MunkiPackageReference[] = [];
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

function parseInstallerChoicesError(value: string) {
  try {
    parseInstallerChoices(value);
    return undefined;
  } catch (error) {
    return error instanceof Error ? error.message : "Installer choices must be valid JSON.";
  }
}

function parseInstallerChoices(value: string): MunkiPackageInstallerChoice[] {
  const text = value.trim();
  if (text === "") return [];
  const parsed = JSON.parse(text) as unknown;
  if (!Array.isArray(parsed)) {
    throw new Error("Installer choices must be a JSON array.");
  }
  return parsed.map((item) => installerChoice(item));
}

function installerChoice(value: unknown): MunkiPackageInstallerChoice {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new Error("Installer choice entries must be JSON objects.");
  }
  const item = value as Record<string, unknown>;
  const choiceIdentifier = stringValue(item.choice_identifier);
  const choiceAttribute = stringValue(item.choice_attribute);
  const attributeSetting = numberValue(item.attribute_setting);
  if (attributeSetting === undefined) {
    throw new Error("Installer choice entries require attribute_setting.");
  }
  return {
    choice_identifier: choiceIdentifier,
    choice_attribute: choiceAttribute,
    attribute_setting: attributeSetting,
  };
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

function stringValue(value: unknown) {
  return typeof value === "string" ? value : "";
}

function numberValue(value: unknown) {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

function hasNoPkgEvidence(form: PackageFormState) {
  return (
    nonEmpty(form.installcheck_script) !== undefined ||
    cleanInstallItems(form.installs).length > 0 ||
    cleanReceipts(form.receipts).length > 0 ||
    form.on_demand
  );
}

function installerTypeLabel(installerType: MunkiInstallerType) {
  switch (installerType) {
    case "pkg":
      return "Package";
    case "copy_from_dmg":
      return "Copy from DMG";
    case "nopkg":
      return "No package";
  }
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

export function toggleArray<T>(values: T[], value: T, enabled: boolean) {
  if (enabled) return Array.from(new Set([...values, value]));
  return values.filter((item) => item !== value);
}

export function packageLabel(pkg: MunkiPackage) {
  return `${pkg.software_name} ${pkg.version}`;
}

function isArchitecture(value: string): value is Architecture {
  return value === "arm64" || value === "x86_64";
}

function rowID() {
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
