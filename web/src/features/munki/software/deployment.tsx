import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import type { MunkiDeploymentHost } from "@lib/api";
import { formatDateTime, formatRelative } from "@lib/utils";

type InstallationStatus = MunkiDeploymentHost["status"];
type MunkiResult = MunkiDeploymentHost["munki_result"];

const INSTALLATION_STATUSES: Record<InstallationStatus, { label: string; description: string }> = {
  installed: {
    label: "Installed",
    description:
      "Based on the configured application detector and the last successful software inventory.",
  },
  not_installed: {
    label: "Not installed",
    description:
      "Based on the configured application detector and the last successful software inventory.",
  },
  unknown: {
    label: "Unknown",
    description: "No detector or successful software inventory is available for this software.",
  },
};

const MUNKI_RESULTS: Record<MunkiResult, { label: string; description: string }> = {
  no_install_needed: {
    label: "No install needed",
    description:
      "The host's current Munki collection indicates no install is needed. Client-side conditions can affect what Munki offers or installs.",
  },
  install_indicated: {
    label: "Install indicated",
    description:
      "The host's current Munki collection indicates an install. Client-side conditions can affect what Munki offers or installs.",
  },
  unresolved: {
    label: "Unresolved",
    description:
      "The host has a current Munki collection, but this item did not resolve to an install result.",
  },
  not_reported: {
    label: "Not reported",
    description: "No current Munki collection is available for this host.",
  },
};

export const INSTALLATION_STATUS_VALUES = [
  "installed",
  "not_installed",
  "unknown",
] as const satisfies readonly InstallationStatus[];

export const INSTALLATION_STATUS_OPTIONS = INSTALLATION_STATUS_VALUES.map((value) => ({
  value,
  label: INSTALLATION_STATUSES[value].label,
}));

export const MUNKI_RESULT_VALUES = [
  "no_install_needed",
  "install_indicated",
  "unresolved",
  "not_reported",
] as const satisfies readonly MunkiResult[];

export const MUNKI_RESULT_OPTIONS = MUNKI_RESULT_VALUES.map((value) => ({
  value,
  label: MUNKI_RESULTS[value].label,
}));

export function InstallationStatusText({ status }: { status: InstallationStatus }) {
  const presentation = INSTALLATION_STATUSES[status];
  const value = <span>{presentation.label}</span>;
  return (
    <Tooltip>
      <TooltipTrigger render={value} />
      <TooltipContent className="max-w-72 text-left">{presentation.description}</TooltipContent>
    </Tooltip>
  );
}

export function MunkiResultText({ result }: { result: MunkiResult }) {
  const presentation = MUNKI_RESULTS[result];
  const value = <span>{presentation.label}</span>;
  return (
    <Tooltip>
      <TooltipTrigger render={value} />
      <TooltipContent className="max-w-72 text-left">{presentation.description}</TooltipContent>
    </Tooltip>
  );
}

export function LastCollected({ value }: { value?: string }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  const relative = <span>{formatRelative(value)}</span>;
  return (
    <Tooltip>
      <TooltipTrigger render={relative} />
      <TooltipContent>{formatDateTime(value)}</TooltipContent>
    </Tooltip>
  );
}
