import type { SantaConfiguration, SantaConfigurationMutation, SantaHostState } from "@/lib/api";
import {
  enumLabel,
  type EnumMetadataMap,
  enumOptions,
  type StatusMetadataMap,
} from "@/lib/enum-metadata";

export type SantaClientMode =
  | SantaHostState["client_mode_reported"]
  | SantaConfiguration["client_mode"];

export type SantaFileAccessAction = NonNullable<
  SantaConfigurationMutation["override_file_access_action"]
>;

type StoredMediaAction = NonNullable<
  NonNullable<SantaConfigurationMutation["removable_media_policy"]>["action"]
>;

export type SantaMediaAction = StoredMediaAction | "none";

export const CLIENT_MODE_VALUES = [
  "monitor",
  "lockdown",
  "standalone",
] as const satisfies readonly NonNullable<SantaConfigurationMutation["client_mode"]>[];

export const CLIENT_MODES = {
  unknown: {
    name: "Unknown",
    description: "The host has not reported a Santa client mode yet.",
    variant: "default",
  },
  monitor: {
    name: "Monitor",
    description: "Santa records decisions but does not block unknown executions.",
    variant: "success",
  },
  lockdown: {
    name: "Lockdown",
    description: "Santa blocks unknown executions unless a matching allow rule applies.",
    variant: "error",
  },
  standalone: {
    name: "Standalone",
    description: "Santa is enforcing local configuration without server-managed sync.",
    variant: "default",
  },
} satisfies StatusMetadataMap<SantaClientMode>;

export const CLIENT_MODE_OPTIONS = CLIENT_MODE_VALUES.map((value) => ({
  value,
  label: CLIENT_MODES[value].name,
}));

export const MEDIA_ACTION_VALUES = [
  "none",
  "allow",
  "block",
  "remount",
] as const satisfies readonly SantaMediaAction[];

export const MEDIA_ACTIONS = {
  none: { name: "No Policy" },
  allow: { name: "Allow" },
  block: { name: "Block" },
  remount: { name: "Remount" },
} satisfies EnumMetadataMap<SantaMediaAction>;

export const MEDIA_ACTION_OPTIONS = enumOptions(MEDIA_ACTIONS, MEDIA_ACTION_VALUES);

export const FILE_ACCESS_ACTION_VALUES = [
  "none",
  "audit_only",
  "disable",
] as const satisfies readonly SantaFileAccessAction[];

export const FILE_ACCESS_ACTIONS = {
  none: { name: "No Override" },
  audit_only: { name: "Audit Only" },
  disable: { name: "Disable" },
} satisfies EnumMetadataMap<SantaFileAccessAction>;

export const FILE_ACCESS_ACTION_OPTIONS = enumOptions(
  FILE_ACCESS_ACTIONS,
  FILE_ACCESS_ACTION_VALUES,
);

export const REMOUNT_FLAG_VALUES = [
  "rdonly",
  "noexec",
  "nosuid",
  "nobrowse",
  "noowners",
  "nodev",
  "async",
  "-j",
] as const;

export type SantaRemountFlag = (typeof REMOUNT_FLAG_VALUES)[number];

export const REMOUNT_FLAG_OPTIONS = REMOUNT_FLAG_VALUES.map((value) => ({
  label: value,
  value,
}));

const mediaActionSet: ReadonlySet<string> = new Set(MEDIA_ACTION_VALUES);
const remountFlagSet: ReadonlySet<string> = new Set(REMOUNT_FLAG_VALUES);

export function isSantaMediaAction(value: string): value is SantaMediaAction {
  return mediaActionSet.has(value);
}

export function isSantaRemountFlag(value: string): value is SantaRemountFlag {
  return remountFlagSet.has(value);
}

export function clientModeLabel(mode: SantaClientMode) {
  return enumLabel(CLIENT_MODES, mode);
}
