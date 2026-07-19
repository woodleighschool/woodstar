import { z } from "zod";

import type { MunkiPackageInstallerChoice } from "@/lib/api";
import { requiredString } from "@/lib/form-validation";
import { isRecord, nonEmpty } from "@/lib/utils";
import {
  MUNKI_INSTALL_ITEM_TYPE_VALUES,
  MUNKI_INSTALLER_TYPE_VALUES,
  MUNKI_RESTART_ACTION_VALUES,
  MUNKI_UNINSTALL_METHOD_VALUES,
} from "@/pages/munki/software/munki-software";

const ARCHITECTURE_VALUES = ["arm64", "x86_64"] as const;
const architectureSet: ReadonlySet<string> = new Set(ARCHITECTURE_VALUES);
const PACKAGE_UNINSTALL_METHOD_VALUES = ["", ...MUNKI_UNINSTALL_METHOD_VALUES] as const;

export type Architecture = (typeof ARCHITECTURE_VALUES)[number];

export const scriptFields = [
  { key: "preinstall_script", label: "Pre-install" },
  { key: "postinstall_script", label: "Post-install" },
  { key: "uninstall_script", label: "Uninstall" },
  { key: "preuninstall_script", label: "Pre-uninstall" },
  { key: "postuninstall_script", label: "Post-uninstall" },
  { key: "version_script", label: "Version" },
  { key: "installcheck_script", label: "Install Check" },
  { key: "uninstallcheck_script", label: "Uninstall Check" },
] as const;

export type ScriptKey = (typeof scriptFields)[number]["key"];

const optionalString = z.string().optional();
const rowIDSchema = z.string().min(1);
const packageReferenceRowSchema = z.object({
  software_id: z.number().int().positive().optional(),
  package_id: z.number().int().positive().optional(),
  software_name: optionalString,
  package_version: optionalString,
  rowID: rowIDSchema,
});
const stringRowSchema = z.object({ rowID: rowIDSchema, value: z.string() });
const installerEnvironmentRowSchema = z.object({
  rowID: rowIDSchema,
  name: z.string(),
  value: z.string(),
});
const installItemRowSchema = z.object({
  rowID: rowIDSchema,
  path: z.string(),
  type: z.enum(MUNKI_INSTALL_ITEM_TYPE_VALUES),
  bundle_identifier: optionalString,
  bundle_name: optionalString,
  bundle_short_version: optionalString,
  bundle_version: optionalString,
  installer_item_location: optionalString,
  md5checksum: optionalString,
  minimum_os_version: optionalString,
  minimum_update_version: optionalString,
  version_comparison_key: optionalString,
});
const receiptRowSchema = z.object({
  rowID: rowIDSchema,
  package_id: z.string(),
  installed_size: z.number().optional(),
  name: optionalString,
  optional: z.boolean().optional(),
  version: optionalString,
});
const itemToCopyRowSchema = z.object({
  rowID: rowIDSchema,
  source_item: z.string(),
  destination_path: z.string(),
  destination_item: optionalString,
  group: optionalString,
  mode: optionalString,
  user: optionalString,
});
const alertSchema = z.object({
  enabled: z.boolean(),
  cancel_label: optionalString,
  detail: optionalString,
  ok_label: optionalString,
  title: optionalString,
});

export type PackageReferenceRow = z.input<typeof packageReferenceRowSchema>;
export type StringRow = z.input<typeof stringRowSchema>;
export type InstallerEnvironmentRow = z.input<typeof installerEnvironmentRowSchema>;
export type InstallItemRow = z.input<typeof installItemRowSchema>;
export type ReceiptRow = z.input<typeof receiptRowSchema>;
export type ItemToCopyRow = z.input<typeof itemToCopyRowSchema>;

const packageFormShape = z.object({
  software_id: z
    .number()
    .int()
    .positive()
    .nullable()
    .transform((value, ctx) => {
      if (value !== null) return value;
      ctx.addIssue({ code: "custom", message: "Software is required." });
      return z.NEVER;
    }),
  version: requiredString("Version"),
  installer_type: z.enum(MUNKI_INSTALLER_TYPE_VALUES),
  installer_object_id: z.number().int().positive().nullable(),
  installer_file: z.custom<File | null>((value) => value === null || value instanceof File),
  uninstallable: z.boolean(),
  uninstall_method: z.enum(PACKAGE_UNINSTALL_METHOD_VALUES),
  restart_required: z.boolean(),
  restart_action: z.enum(MUNKI_RESTART_ACTION_VALUES),
  minimum_munki_version: z.string(),
  minimum_os_version: z.string(),
  maximum_os_version: z.string(),
  supported_architectures: z.array(z.enum(ARCHITECTURE_VALUES)),
  blocking_applications_none: z.boolean(),
  blocking_applications: z.array(stringRowSchema),
  installable_condition: z.string(),
  blocking_applications_manual_quit_only: z.boolean(),
  blocking_applications_quit_script: z.string(),
  requires: z.array(packageReferenceRowSchema),
  update_for: z.array(packageReferenceRowSchema),
  unattended_install: z.boolean(),
  unattended_uninstall: z.boolean(),
  on_demand: z.boolean(),
  precache: z.boolean(),
  autoremove: z.boolean(),
  apple_item: z.boolean(),
  suppress_bundle_relocation: z.boolean(),
  force_install_after_date: z.string(),
  installed_size: z.string(),
  package_path: z.string(),
  installer_choices_xml: z.string(),
  installer_environment: z.array(installerEnvironmentRowSchema),
  installs: z.array(installItemRowSchema),
  receipts: z.array(receiptRowSchema),
  items_to_copy: z.array(itemToCopyRowSchema),
  notes: z.string(),
  installcheck_script: z.string(),
  uninstallcheck_script: z.string(),
  preinstall_script: z.string(),
  postinstall_script: z.string(),
  preuninstall_script: z.string(),
  postuninstall_script: z.string(),
  uninstall_script: z.string(),
  version_script: z.string(),
  preinstall_alert: alertSchema,
  preuninstall_alert: alertSchema,
});

export function packageFormSchema() {
  return packageFormShape.superRefine((form, ctx) => {
    if (form.uninstallable && form.uninstall_method === "") {
      ctx.addIssue({
        code: "custom",
        message: "Uninstallable packages require an uninstall method.",
        path: ["uninstall_method"],
      });
    }
    if (
      form.installer_type !== "nopkg" &&
      form.installer_object_id === null &&
      form.installer_file === null
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Installer file is required.",
        path: ["installer_file"],
      });
    }
    if (form.installed_size.trim() !== "") {
      const installedSize = Number(form.installed_size);
      if (!Number.isSafeInteger(installedSize) || installedSize < 0) {
        ctx.addIssue({
          code: "custom",
          message: "Installed size must be a non-negative whole number.",
          path: ["installed_size"],
        });
      }
    }
    if (
      form.force_install_after_date.trim() !== "" &&
      Number.isNaN(new Date(form.force_install_after_date).valueOf())
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Force install after must be a valid date and time.",
        path: ["force_install_after_date"],
      });
    }
    const environmentNames = new Set<string>();
    for (const variable of form.installer_environment) {
      const name = variable.name.trim();
      if (name === "") {
        ctx.addIssue({
          code: "custom",
          message: "Every installer environment variable requires a name.",
          path: ["installer_environment"],
        });
        break;
      }
      if (environmentNames.has(name)) {
        ctx.addIssue({
          code: "custom",
          message: `Installer environment contains duplicate name ${name}.`,
          path: ["installer_environment"],
        });
        break;
      }
      environmentNames.add(name);
    }
    if (form.installs.some((item) => item.path.trim() === "")) {
      ctx.addIssue({
        code: "custom",
        message: "Every install item requires a path.",
        path: ["installs"],
      });
    }
    if (
      form.receipts.some(
        (receipt) =>
          receipt.package_id.trim() === "" ||
          (receipt.installed_size !== undefined &&
            (!Number.isSafeInteger(receipt.installed_size) || receipt.installed_size < 0)),
      )
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Every receipt requires a package ID and a non-negative whole installed size.",
        path: ["receipts"],
      });
    }
    if (
      form.items_to_copy.some(
        (item) => item.source_item.trim() === "" || item.destination_path.trim() === "",
      )
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Every copy item requires a source item and destination path.",
        path: ["items_to_copy"],
      });
    }
    if (form.requires.some((reference) => reference.software_id === undefined)) {
      ctx.addIssue({
        code: "custom",
        message: "Every requirement must select a package.",
        path: ["requires"],
      });
    }
    if (form.update_for.some((reference) => reference.software_id === undefined)) {
      ctx.addIssue({
        code: "custom",
        message: "Every update target must select a package.",
        path: ["update_for"],
      });
    }
    if (
      !form.blocking_applications_none &&
      form.blocking_applications.some((application) => application.value.trim() === "")
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Every blocking application requires a name.",
        path: ["blocking_applications"],
      });
    }
    if (form.installer_type === "copy_from_dmg" && !hasItemToCopyInput(form.items_to_copy)) {
      ctx.addIssue({
        code: "custom",
        message: "Copy from DMG packages require at least one item to copy.",
        path: ["items_to_copy"],
      });
    }
    if (form.installer_type !== "nopkg") {
      const installerChoicesError = parseInstallerChoicesError(form.installer_choices_xml);
      if (installerChoicesError) {
        ctx.addIssue({
          code: "custom",
          message: installerChoicesError,
          path: ["installer_choices_xml"],
        });
      }
    }
    if (
      form.uninstallable &&
      form.uninstall_method === "removepackages" &&
      !hasReceiptInput(form.receipts)
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Remove packages uninstall requires at least one receipt.",
        path: ["receipts"],
      });
    }
    if (
      form.uninstallable &&
      form.uninstall_method === "remove_copied_items" &&
      !hasItemToCopyInput(form.items_to_copy)
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Remove copied items uninstall requires at least one item to copy.",
        path: ["items_to_copy"],
      });
    }
    if (
      form.uninstallable &&
      form.uninstall_method === "uninstall_script" &&
      nonEmpty(form.uninstall_script) === undefined
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Uninstall script method requires an uninstall script.",
        path: ["uninstall_script"],
      });
    }
  });
}

export type PackageFormInput = z.input<ReturnType<typeof packageFormSchema>>;
export type PackageFormOutput = z.output<ReturnType<typeof packageFormSchema>>;

export function parseInstallerChoices(value: string): MunkiPackageInstallerChoice[] {
  const text = value.trim();
  if (text === "") return [];
  const parsed = JSON.parse(text) as unknown;
  if (!Array.isArray(parsed)) {
    throw new Error("Installer choices must be a JSON array.");
  }
  return parsed.map((item) => installerChoice(item));
}

export function isArchitecture(value: string): value is Architecture {
  return architectureSet.has(value);
}

function parseInstallerChoicesError(value: string) {
  try {
    parseInstallerChoices(value);
    return undefined;
  } catch (error) {
    return error instanceof Error ? error.message : "Installer choices must be valid JSON.";
  }
}

function installerChoice(value: unknown): MunkiPackageInstallerChoice {
  if (!isRecord(value)) {
    throw new Error("Installer choice entries must be JSON objects.");
  }
  const choiceIdentifier = stringValue(value.choice_identifier).trim();
  const choiceAttribute = stringValue(value.choice_attribute);
  const attributeSetting = numberValue(value.attribute_setting);
  if (choiceIdentifier === "") {
    throw new Error("Installer choice entries require choice_identifier.");
  }
  if (
    attributeSetting === undefined ||
    !Number.isInteger(attributeSetting) ||
    attributeSetting < -2_147_483_648 ||
    attributeSetting > 2_147_483_647
  ) {
    throw new Error("Installer choice entries require a whole int32 attribute_setting.");
  }
  return {
    choice_identifier: choiceIdentifier,
    choice_attribute: choiceAttribute,
    attribute_setting: attributeSetting,
  };
}

function hasReceiptInput(rows: ReceiptRow[]) {
  return rows.some((row) => row.package_id.trim() !== "");
}

function hasItemToCopyInput(rows: ItemToCopyRow[]) {
  return rows.some((row) => row.source_item.trim() !== "" || row.destination_path.trim() !== "");
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
