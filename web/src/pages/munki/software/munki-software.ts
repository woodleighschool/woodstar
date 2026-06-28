import type { MunkiInclude, MunkiPackageInstallItem, MunkiPackageMutation } from "@/lib/api";
import { type EnumMetadataMap, enumOptions } from "@/lib/enum-metadata";

export type MunkiInstallerType = NonNullable<MunkiPackageMutation["installer_type"]>;
export type MunkiRestartAction = NonNullable<MunkiPackageMutation["restart_action"]>;
export type MunkiUninstallMethod = NonNullable<MunkiPackageMutation["uninstall_method"]>;
export type MunkiInstallItemType = MunkiPackageInstallItem["type"];
export type MunkiSoftwareAction = MunkiInclude["actions"][number];
export type MunkiPackageStrategy = MunkiInclude["package"]["strategy"];

function rawEnumOptions<T extends string>(values: readonly T[]) {
  return values.map((value) => ({ value, label: value }));
}

export const MUNKI_INSTALLER_TYPE_OPTIONS = rawEnumOptions<MunkiInstallerType>([
  "pkg",
  "nopkg",
  "copy_from_dmg",
]);

export const MUNKI_RESTART_ACTION_OPTIONS = rawEnumOptions<MunkiRestartAction>([
  "RequireLogout",
  "RecommendRestart",
  "RequireRestart",
  "RequireShutdown",
]);

export const MUNKI_UNINSTALL_METHOD_OPTIONS = rawEnumOptions<MunkiUninstallMethod>([
  "removepackages",
  "remove_copied_items",
  "uninstall_script",
]);

export const MUNKI_INSTALL_ITEM_TYPE_OPTIONS = rawEnumOptions<MunkiInstallItemType>([
  "application",
  "bundle",
  "plist",
  "file",
]);

export const MUNKI_SOFTWARE_ACTION_VALUES = [
  "managed_installs",
  "managed_uninstalls",
  "managed_updates",
  "optional_installs",
  "featured_items",
  "default_installs",
] as const satisfies readonly MunkiSoftwareAction[];

export const MUNKI_SOFTWARE_ACTIONS = {
  managed_installs: {
    name: "Managed Installs",
    description: "Installs the item on matching hosts and keeps it installed.",
  },
  managed_uninstalls: {
    name: "Managed Uninstalls",
    description: "Removes the item from matching hosts and keeps it uninstalled.",
  },
  managed_updates: {
    name: "Managed Updates",
    description: "Updates the item only on hosts where it is already installed.",
  },
  optional_installs: {
    name: "Optional Installs",
    description: "Makes the item available in Managed Software Center.",
  },
  featured_items: {
    name: "Featured Items",
    description: "Highlights the item in Managed Software Center.",
  },
  default_installs: {
    name: "Default Installs",
    description: "Installs the item by default; users can remove it in Managed Software Center.",
  },
} satisfies EnumMetadataMap<MunkiSoftwareAction>;

export const MUNKI_SOFTWARE_ACTION_OPTIONS = enumOptions(MUNKI_SOFTWARE_ACTIONS);

// An include resolves to one primary intent plus, for optional installs, the
// featured/default modifiers. These subsets drive the radio + checkbox editor.
export const MUNKI_PRIMARY_ACTION_VALUES = [
  "managed_installs",
  "managed_uninstalls",
  "managed_updates",
  "optional_installs",
] as const satisfies readonly MunkiSoftwareAction[];

export type MunkiPrimaryAction = (typeof MUNKI_PRIMARY_ACTION_VALUES)[number];

export const MUNKI_OPTIONAL_MODIFIER_VALUES = [
  "featured_items",
  "default_installs",
] as const satisfies readonly MunkiSoftwareAction[];

export const MUNKI_PACKAGE_STRATEGY_VALUES = [
  "latest",
  "specific",
] as const satisfies readonly MunkiPackageStrategy[];
