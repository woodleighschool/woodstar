import type { MunkiInclude } from "@/lib/api";
import { type EnumMetadataMap, enumOptions } from "@/lib/enum-metadata";

export type MunkiSoftwareAction = MunkiInclude["actions"][number];

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

export const MUNKI_SOFTWARE_ACTION_OPTIONS = enumOptions(
  MUNKI_SOFTWARE_ACTIONS,
  MUNKI_SOFTWARE_ACTION_VALUES,
);

// Managed installs and uninstalls are exclusive assignment intents. Managed
// updates and optional installs can be combined for optional-but-forced updates.
export const MUNKI_ASSIGNMENT_ACTION_VALUES = [
  "managed_installs",
  "managed_uninstalls",
  "managed_updates",
  "optional_installs",
] as const satisfies readonly MunkiSoftwareAction[];

export const MUNKI_OPTIONAL_PRESENTATION_ACTION_VALUES = [
  "featured_items",
  "default_installs",
] as const satisfies readonly MunkiSoftwareAction[];
