import type { MunkiPackageMutation } from "@/hooks/use-munki-packages";
import type { PackageInstallItem, SoftwareInclude } from "@/lib/api";
import { type EnumMetadataMap, enumOptions } from "@/lib/enum-metadata";

export type MunkiInstallerType = NonNullable<MunkiPackageMutation["installer_type"]>;
export type MunkiRestartAction = NonNullable<MunkiPackageMutation["restart_action"]>;
export type MunkiUninstallMethod = NonNullable<MunkiPackageMutation["uninstall_method"]>;
export type MunkiInstallItemType = PackageInstallItem["type"];
export type MunkiSoftwareAction = SoftwareInclude["actions"][number];
export type MunkiPackageStrategy = SoftwareInclude["package"]["strategy"];

function rawEnumOptions<T extends string>(values: readonly T[]) {
  return values.map((value) => ({ value, label: value }));
}

export const MUNKI_INSTALLER_TYPE_OPTIONS = rawEnumOptions<MunkiInstallerType>([
  "pkg",
  "nopkg",
  "copy_from_dmg",
]);

export const MUNKI_RESTART_ACTION_OPTIONS = rawEnumOptions<MunkiRestartAction>([
  "None",
  "RequireLogout",
  "RecommendRestart",
  "RequireRestart",
  "RequireShutdown",
]);

export const MUNKI_UNINSTALL_METHOD_OPTIONS = rawEnumOptions<MunkiUninstallMethod>([
  "none",
  "removepackages",
  "remove_copied_items",
  "uninstall_script",
  "uninstall_package",
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
    description: "Forces installation by writing managed_installs.",
  },
  managed_uninstalls: {
    name: "Managed Uninstalls",
    description: "Forces removal by writing managed_uninstalls.",
  },
  managed_updates: {
    name: "Managed Updates",
    description: "Updates installed items by writing managed_updates.",
  },
  optional_installs: {
    name: "Optional Installs",
    description: "Makes the item available in Managed Software Center.",
  },
  featured_items: {
    name: "Featured Items",
    description: "Highlights the item in Managed Software Center when it is also optional.",
  },
  default_installs: {
    name: "Default Installs",
    description: "Preselects the optional item for new self-service manifests.",
  },
} satisfies EnumMetadataMap<MunkiSoftwareAction>;

export const MUNKI_SOFTWARE_ACTION_OPTIONS = enumOptions(MUNKI_SOFTWARE_ACTIONS);

export const MUNKI_PACKAGE_STRATEGY_VALUES = [
  "latest",
  "specific",
] as const satisfies readonly MunkiPackageStrategy[];
