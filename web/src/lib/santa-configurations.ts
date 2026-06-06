import type { SantaClientMode, SantaConfigurationMutation } from "@/hooks/use-santa";
import { enumLabel, enumOptions, type EnumMetadataMap } from "@/lib/enum-metadata";

type StoredMediaAction = NonNullable<NonNullable<SantaConfigurationMutation["removable_media_policy"]>["action"]>;

export type SantaMediaAction = StoredMediaAction | "none";

export const CLIENT_MODE_VALUES = ["monitor", "lockdown", "standalone"] as const satisfies readonly NonNullable<
  SantaConfigurationMutation["client_mode"]
>[];

export const CLIENT_MODES = {
  unknown: {
    name: "Unknown",
    description: "The host has not reported a Santa client mode yet.",
    variant: "secondary",
  },
  monitor: {
    name: "Monitor",
    description: "Santa records decisions but does not block unknown executions.",
    variant: "success",
  },
  lockdown: {
    name: "Lockdown",
    description: "Santa blocks unknown executions unless a matching allow rule applies.",
    variant: "destructive",
  },
  standalone: {
    name: "Standalone",
    description: "Santa is enforcing local configuration without server-managed sync.",
    variant: "secondary",
  },
} satisfies EnumMetadataMap<SantaClientMode>;

export const CLIENT_MODE_OPTIONS = enumOptions(CLIENT_MODES).filter((option) =>
  CLIENT_MODE_VALUES.includes(option.value as NonNullable<SantaConfigurationMutation["client_mode"]>),
);

export const MEDIA_ACTION_VALUES = ["none", "allow", "block", "remount"] as const satisfies readonly SantaMediaAction[];

export const MEDIA_ACTIONS = {
  none: { name: "No Policy" },
  allow: { name: "Allow" },
  block: { name: "Block" },
  remount: { name: "Remount" },
} satisfies EnumMetadataMap<SantaMediaAction>;

export const MEDIA_ACTION_OPTIONS = enumOptions(MEDIA_ACTIONS);

export function clientModeLabel(mode: string | null | undefined) {
  return enumLabel(CLIENT_MODES, mode);
}

export function mediaActionLabel(action: string | null | undefined) {
  return enumLabel(MEDIA_ACTIONS, action);
}
