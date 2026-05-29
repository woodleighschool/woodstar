import type { SantaClientMode, SantaConfigurationMutation } from "@/hooks/use-santa";
import { enumLabel, enumOptions, type EnumMetadataMap } from "@/lib/enum-metadata";

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

export function clientModeLabel(mode: string | null | undefined) {
  return enumLabel(CLIENT_MODES, mode);
}
