import type { MunkiAssignmentMutation } from "@/hooks/munki/assignments";
import type { MunkiPackageMutation } from "@/hooks/munki/packages";
import type { PackageInstallItem } from "@/lib/api-client/types.gen";
import { enumLabel, enumOptions, type EnumMetadataMap } from "@/lib/enum-metadata";

export type MunkiInstallerType = NonNullable<MunkiPackageMutation["installer_type"]>;
export type MunkiRestartAction = NonNullable<MunkiPackageMutation["restart_action"]>;
export type MunkiUninstallMethod = NonNullable<MunkiPackageMutation["uninstall_method"]>;
export type MunkiInstallItemType = PackageInstallItem["type"];
export type MunkiAssignmentAction = NonNullable<MunkiAssignmentMutation["action"]>;
export type MunkiAssignmentEffect = MunkiAssignmentMutation["effect"];
export type MunkiPackageSelection = NonNullable<MunkiAssignmentMutation["package_selection"]>;

export const MUNKI_INSTALLER_TYPE_VALUES = [
  "pkg",
  "nopkg",
  "copy_from_dmg",
  "profile",
  "apple_update_metadata",
  "startosinstall",
  "stage_os_installer",
] as const satisfies readonly MunkiInstallerType[];

export const MUNKI_INSTALLER_TYPES = {
  pkg: { name: "Package" },
  nopkg: { name: "No package" },
  copy_from_dmg: { name: "Copy from DMG" },
  profile: { name: "Profile" },
  apple_update_metadata: { name: "Apple update metadata" },
  startosinstall: { name: "Start OS install" },
  stage_os_installer: { name: "Stage OS installer" },
} satisfies EnumMetadataMap<MunkiInstallerType>;

export const MUNKI_INSTALLER_TYPE_OPTIONS = enumOptions(MUNKI_INSTALLER_TYPES);

export const MUNKI_RESTART_ACTION_VALUES = [
  "None",
  "RequireLogout",
  "RecommendRestart",
  "RequireRestart",
  "RequireShutdown",
] as const satisfies readonly MunkiRestartAction[];

export const MUNKI_RESTART_ACTIONS = {
  None: { name: "None" },
  RequireLogout: { name: "Require logout" },
  RecommendRestart: { name: "Recommend restart" },
  RequireRestart: { name: "Require restart" },
  RequireShutdown: { name: "Require shutdown" },
} satisfies EnumMetadataMap<MunkiRestartAction>;

export const MUNKI_RESTART_ACTION_OPTIONS = enumOptions(MUNKI_RESTART_ACTIONS);

export const MUNKI_UNINSTALL_METHOD_VALUES = [
  "none",
  "removepackages",
  "remove_copied_items",
  "remove_profile",
  "remove_app",
  "uninstall_script",
  "uninstall_package",
  "custom",
] as const satisfies readonly MunkiUninstallMethod[];

export const MUNKI_UNINSTALL_METHODS = {
  none: { name: "None" },
  removepackages: { name: "Remove packages" },
  remove_copied_items: { name: "Remove copied items" },
  remove_profile: { name: "Remove profile" },
  remove_app: { name: "Remove app" },
  uninstall_script: { name: "Uninstall script" },
  uninstall_package: { name: "Uninstall package" },
  custom: { name: "Custom" },
} satisfies EnumMetadataMap<MunkiUninstallMethod>;

export const MUNKI_UNINSTALL_METHOD_OPTIONS = enumOptions(MUNKI_UNINSTALL_METHODS);

export const MUNKI_INSTALL_ITEM_TYPE_VALUES = [
  "application",
  "bundle",
  "plist",
  "file",
] as const satisfies readonly MunkiInstallItemType[];

export const MUNKI_INSTALL_ITEM_TYPES = {
  application: { name: "Application" },
  bundle: { name: "Bundle" },
  plist: { name: "Plist" },
  file: { name: "File" },
} satisfies EnumMetadataMap<MunkiInstallItemType>;

export const MUNKI_INSTALL_ITEM_TYPE_OPTIONS = enumOptions(MUNKI_INSTALL_ITEM_TYPES);

export const MUNKI_ASSIGNMENT_EFFECT_VALUES = [
  "include",
  "exclude",
] as const satisfies readonly MunkiAssignmentEffect[];

export const MUNKI_ASSIGNMENT_EFFECTS = {
  include: {
    name: "Include",
    description: "Adds matching hosts to this software assignment.",
  },
  exclude: {
    name: "Exclude",
    description: "Stops matching hosts from receiving this software.",
  },
} satisfies EnumMetadataMap<MunkiAssignmentEffect>;

export const MUNKI_ASSIGNMENT_EFFECT_OPTIONS = enumOptions(MUNKI_ASSIGNMENT_EFFECTS);

export const MUNKI_ASSIGNMENT_ACTION_VALUES = [
  "install",
  "remove",
  "update_if_present",
  "none",
] as const satisfies readonly MunkiAssignmentAction[];

export const MUNKI_ASSIGNMENT_ACTIONS = {
  install: {
    name: "Managed Installs",
    description: "Forces installation by writing managed_installs.",
  },
  remove: {
    name: "Managed Uninstalls",
    description: "Forces removal by writing managed_uninstalls.",
  },
  update_if_present: {
    name: "Managed Updates",
    description: "Updates installed items by writing managed_updates.",
  },
  none: {
    name: "None",
    description: "Only Optional Installs and Featured Items section membership is rendered.",
  },
} satisfies EnumMetadataMap<MunkiAssignmentAction>;

export const MUNKI_ASSIGNMENT_ACTION_OPTIONS = enumOptions(MUNKI_ASSIGNMENT_ACTIONS);

export const MUNKI_PACKAGE_SELECTION_VALUES = [
  "latest_eligible",
  "specific_package",
] as const satisfies readonly MunkiPackageSelection[];

export const MUNKI_PACKAGE_SELECTIONS = {
  latest_eligible: {
    name: "Latest compatible",
    description: "Render the Munki name and include all eligible pkginfos for the client to choose from.",
  },
  specific_package: {
    name: "Pinned package",
    description: "Render Name--Version and include only that pkginfo candidate.",
  },
} satisfies EnumMetadataMap<MunkiPackageSelection>;

export const MUNKI_PACKAGE_SELECTION_OPTIONS = enumOptions(MUNKI_PACKAGE_SELECTIONS);

export function munkiInstallerTypeLabel(value: string | null | undefined) {
  return enumLabel(MUNKI_INSTALLER_TYPES, value);
}

export function munkiRestartActionLabel(value: string | null | undefined) {
  return enumLabel(MUNKI_RESTART_ACTIONS, value);
}

export function munkiAssignmentEffectLabel(value: string | null | undefined) {
  return enumLabel(MUNKI_ASSIGNMENT_EFFECTS, value);
}

export function munkiAssignmentActionLabel(value: string | null | undefined) {
  if (!value) return "None";
  return enumLabel(MUNKI_ASSIGNMENT_ACTIONS, value);
}

export function munkiPackageSelectionLabel(value: string | null | undefined) {
  if (!value) return enumLabel(MUNKI_PACKAGE_SELECTIONS, "latest_eligible");
  return enumLabel(MUNKI_PACKAGE_SELECTIONS, value);
}

export function munkiAssignmentActionDescription(value: MunkiAssignmentAction) {
  return MUNKI_ASSIGNMENT_ACTIONS[value].description;
}

export function munkiPackageSelectionDescription(value: MunkiPackageSelection) {
  return MUNKI_PACKAGE_SELECTIONS[value].description;
}
