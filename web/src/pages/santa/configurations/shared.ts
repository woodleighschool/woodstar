import type { SantaConfiguration, SantaConfigurationMutation } from "@/hooks/use-santa";

export const CLIENT_MODE_OPTIONS: { value: NonNullable<SantaConfigurationMutation["client_mode"]>; label: string }[] = [
  { value: "monitor", label: "Monitor" },
  { value: "lockdown", label: "Lockdown" },
  { value: "standalone", label: "Standalone" },
];

export function clientModeLabel(mode: SantaConfiguration["client_mode"]) {
  return CLIENT_MODE_OPTIONS.find((option) => option.value === mode)?.label ?? mode;
}
