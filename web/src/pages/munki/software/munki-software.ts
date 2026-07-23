import type { MunkiInclude, MunkiPackageInstallItem, MunkiPackageMutation } from "@/lib/api";
import { type EnumMetadataMap, enumOptions } from "@/lib/enum-metadata";

export type MunkiInstallerType = NonNullable<MunkiPackageMutation["installer_type"]>;
export type MunkiRestartAction = NonNullable<MunkiPackageMutation["restart_action"]>;
export type MunkiUninstallMethod = NonNullable<MunkiPackageMutation["uninstall_method"]>;
export type MunkiInstallItemType = MunkiPackageInstallItem["type"];
export type MunkiPackageStrategy = MunkiInclude["package"]["strategy"];

function rawEnumOptions<T extends string>(values: readonly T[]) {
  return values.map((value) => ({ value, label: value }));
}

export const MUNKI_INSTALLER_TYPE_VALUES = [
  "pkg",
  "nopkg",
  "copy_from_dmg",
] as const satisfies readonly MunkiInstallerType[];

export const MUNKI_INSTALLER_TYPES = {
  pkg: {
    name: "pkg",
    description: "Installs an Apple installer package.",
  },
  nopkg: {
    name: "nopkg",
    description: "Uses package metadata and scripts without an installer payload.",
  },
  copy_from_dmg: {
    name: "copy_from_dmg",
    description: "Copies the configured items from a mounted disk image.",
  },
} satisfies EnumMetadataMap<MunkiInstallerType>;

export const MUNKI_INSTALLER_TYPE_OPTIONS = enumOptions(
  MUNKI_INSTALLER_TYPES,
  MUNKI_INSTALLER_TYPE_VALUES,
);

export const MUNKI_RESTART_ACTION_VALUES = [
  "RequireLogout",
  "RecommendRestart",
  "RequireRestart",
  "RequireShutdown",
] as const satisfies readonly MunkiRestartAction[];

export const MUNKI_RESTART_ACTION_OPTIONS = rawEnumOptions(MUNKI_RESTART_ACTION_VALUES);

export const MUNKI_UNINSTALL_METHOD_VALUES = [
  "removepackages",
  "remove_copied_items",
  "uninstall_script",
] as const satisfies readonly MunkiUninstallMethod[];

export const MUNKI_UNINSTALL_METHOD_OPTIONS = rawEnumOptions(MUNKI_UNINSTALL_METHOD_VALUES);

export const MUNKI_INSTALL_ITEM_TYPE_VALUES = [
  "application",
  "bundle",
  "plist",
  "file",
] as const satisfies readonly MunkiInstallItemType[];

export const MUNKI_INSTALL_ITEM_TYPE_OPTIONS = rawEnumOptions(MUNKI_INSTALL_ITEM_TYPE_VALUES);

export const MUNKI_PACKAGE_STRATEGY_VALUES = [
  "latest",
  "specific",
] as const satisfies readonly MunkiPackageStrategy[];
