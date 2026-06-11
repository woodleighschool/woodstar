import type { MunkiPackageMutation } from "@/hooks/use-munki-packages";
import type { PackageInstallItem, SoftwareInclude } from "@/lib/api";
import { enumLabel, enumOptions, type EnumMetadataMap } from "@/lib/enum-metadata";

export type MunkiInstallerType = NonNullable<MunkiPackageMutation["installer_type"]>;
export type MunkiRestartAction = NonNullable<MunkiPackageMutation["restart_action"]>;
export type MunkiUninstallMethod = NonNullable<MunkiPackageMutation["uninstall_method"]>;
export type MunkiInstallItemType = PackageInstallItem["type"];
export type MunkiSoftwareAction = SoftwareInclude["actions"][number];
export type MunkiPackageStrategy = SoftwareInclude["package"]["strategy"];

export const MUNKI_INSTALLER_TYPES = {
  pkg: { name: "Package" },
  nopkg: { name: "No package" },
  copy_from_dmg: { name: "Copy from DMG" },
} satisfies EnumMetadataMap<MunkiInstallerType>;

export const MUNKI_INSTALLER_TYPE_OPTIONS = enumOptions(MUNKI_INSTALLER_TYPES);

export const MUNKI_RESTART_ACTIONS = {
  None: { name: "None" },
  RequireLogout: { name: "Require logout" },
  RecommendRestart: { name: "Recommend restart" },
  RequireRestart: { name: "Require restart" },
  RequireShutdown: { name: "Require shutdown" },
} satisfies EnumMetadataMap<MunkiRestartAction>;

export const MUNKI_RESTART_ACTION_OPTIONS = enumOptions(MUNKI_RESTART_ACTIONS);

export const MUNKI_UNINSTALL_METHODS = {
  none: { name: "None" },
  removepackages: { name: "Remove packages" },
  remove_copied_items: { name: "Remove copied items" },
  uninstall_script: { name: "Uninstall script" },
  uninstall_package: { name: "Uninstall package" },
} satisfies EnumMetadataMap<MunkiUninstallMethod>;

export const MUNKI_UNINSTALL_METHOD_OPTIONS = enumOptions(MUNKI_UNINSTALL_METHODS);

export const MUNKI_INSTALL_ITEM_TYPES = {
  application: { name: "Application" },
  bundle: { name: "Bundle" },
  plist: { name: "Plist" },
  file: { name: "File" },
} satisfies EnumMetadataMap<MunkiInstallItemType>;

export const MUNKI_INSTALL_ITEM_TYPE_OPTIONS = enumOptions(MUNKI_INSTALL_ITEM_TYPES);

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

export const MUNKI_PACKAGE_STRATEGY_VALUES = ["latest", "specific"] as const satisfies readonly MunkiPackageStrategy[];

export function munkiInstallerTypeLabel(value: string | null | undefined) {
  return enumLabel(MUNKI_INSTALLER_TYPES, value);
}
