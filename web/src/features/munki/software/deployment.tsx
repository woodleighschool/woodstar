import { Badge } from "@components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import type { MunkiDeploymentHost, MunkiHostManifestSoftware } from "@lib/api";

type MunkiDeploymentStatus = NonNullable<MunkiHostManifestSoftware["status"]>;

const STATUSES = {
  up_to_date: {
    label: "Up to date",
    description: "Munki reports the managed install or update is installed with nothing pending.",
    variant: "success",
  },
  pending: {
    label: "Pending",
    description: "Munki reports that this software still has a version to install.",
    variant: "warning",
  },
  not_installed: {
    label: "Not installed",
    description: "The current report does not include this managed software item.",
    variant: "error",
  },
  installed: {
    label: "Installed",
    description: "The current report shows this optional software is installed.",
    variant: "success",
  },
  available: {
    label: "Available",
    description: "This optional software is available to the host.",
    variant: "info",
  },
} as const;

export const MUNKI_DEPLOYMENT_STATUS_VALUES = [
  "up_to_date",
  "pending",
  "not_installed",
  "installed",
  "available",
] as const satisfies readonly NonNullable<MunkiDeploymentHost["status"]>[];

export const MUNKI_DEPLOYMENT_STATUS_OPTIONS = MUNKI_DEPLOYMENT_STATUS_VALUES.map((value) => ({
  value,
  label: STATUSES[value].label,
}));

export const MUNKI_REPORT_STATES = {
  current: {
    name: "Current",
    description: "The latest Munki report is available for this host.",
  },
  not_contacted: {
    name: "Not contacted",
    description: "Woodstar has not received a Munki request from this host.",
    variant: "outline",
  },
  never_collected: {
    name: "Never collected",
    description: "Woodstar has not collected a complete Munki report from this host.",
    variant: "outline",
  },
  no_report: {
    name: "No report",
    description: "The latest complete collection did not contain a Munki report.",
    variant: "warning",
  },
  collection_failed: {
    name: "Collection failed",
    description: "The latest Munki collection failed.",
    variant: "error",
  },
} as const;

export function deploymentStatusName(status: MunkiDeploymentStatus) {
  return STATUSES[status].label;
}

export function DeploymentStatusBadge({ status }: { status: MunkiDeploymentStatus }) {
  const presentation = STATUSES[status];
  const badge = <Badge variant={presentation.variant}>{presentation.label}</Badge>;
  return (
    <Tooltip>
      <TooltipTrigger render={badge} />
      <TooltipContent className="max-w-72 text-left">{presentation.description}</TooltipContent>
    </Tooltip>
  );
}
