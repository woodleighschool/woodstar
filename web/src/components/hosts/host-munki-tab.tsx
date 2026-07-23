import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import {
  CircleCheck,
  Clock3,
  Download,
  PackageOpen,
  RefreshCw,
  Star,
  Trash2,
  type LucideIcon,
} from "lucide-react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { QueryError } from "@/components/query-error";
import { SoftwareArtwork } from "@/components/software/software-icon";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useHostMunkiSoftware } from "@/hooks/use-hosts";
import type { ApiError, MunkiHostManifestSoftware, MunkiHostState } from "@/lib/api";
import { MUNKI_SOFTWARE_ACTIONS, type MunkiSoftwareAction } from "@/lib/munki-software-actions";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

const actionIcons: Record<MunkiSoftwareAction, LucideIcon> = {
  managed_installs: Download,
  managed_uninstalls: Trash2,
  managed_updates: RefreshCw,
  optional_installs: PackageOpen,
  featured_items: Star,
  default_installs: Download,
};

const softwareColumns: ColumnDef<MunkiHostManifestSoftware>[] = [
  {
    id: "software",
    accessorFn: (row) => row.software.name,
    header: () => "Software",
    cell: ({ row }) => <MunkiSoftwareCell software={row.original} />,
  },
  {
    accessorKey: "actions",
    header: () => "Actions",
    cell: ({ row }) => (
      <div className="flex flex-wrap gap-1">
        {row.original.actions.map((action) => (
          <MunkiActionBadge key={action} action={action} />
        ))}
      </div>
    ),
  },
  {
    id: "target_version",
    header: () => "Target Version",
    cell: ({ row }) => row.original.observation?.target_version,
  },
  {
    id: "status",
    header: () => "Status",
    cell: ({ row }) => <MunkiSoftwareStatusBadge software={row.original} />,
  },
];

interface HostMunkiTabProps {
  hostId: number;
  munki: MunkiHostState | null | undefined;
  stateError: ApiError | null;
  onStateRetry: () => void;
}

export function HostMunkiTab({ hostId, munki, stateError, onStateRetry }: HostMunkiTabProps) {
  const software = useHostMunkiSoftware(hostId, { per_page: MAX_PAGE_SIZE });
  const problems = munki
    ? [
        ...problemRows("Errors", munki.errors),
        ...problemRows("Warnings", munki.warnings),
        ...problemRows("Problem Installs", munki.problem_installs),
      ]
    : [];

  return (
    <div className="flex flex-col gap-4">
      {stateError ? (
        <QueryError title="Failed to load Munki state" error={stateError} onRetry={onStateRetry} />
      ) : munki === null ? (
        <EmptyPanel>No Munki run reported</EmptyPanel>
      ) : munki ? (
        <Card>
          <CardContent>
            <KeyValueGrid>
              <KeyValueItem label="Version" value={munki.version} />
              <KeyValueItem label="Manifest" value={munki.manifest_name} />
              <KeyValueItem label="Status" value={<MunkiStatusBadge munki={munki} />} />
              <KeyValueItem label="Last Run Started" value={formatRelative(munki.run_started_at)} />
              <KeyValueItem label="Last Run Ended" value={formatRelative(munki.run_ended_at)} />
            </KeyValueGrid>
          </CardContent>
        </Card>
      ) : null}

      {problems.length > 0 ? (
        <Card className="gap-4 py-4">
          <CardHeader>
            <CardTitle>Problems</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              {problems.map((problem) => (
                <div
                  key={`${problem.kind}-${problem.value}`}
                  className="flex min-w-0 items-start gap-3"
                >
                  <Badge variant="secondary" className="shrink-0">
                    {problem.kind}
                  </Badge>
                  <span className="min-w-0 text-sm wrap-break-word">{problem.value}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : null}

      <Card className="gap-4 py-4">
        <CardHeader>
          <CardTitle>Software</CardTitle>
        </CardHeader>
        <CardContent>
          {software.error ? (
            <QueryError
              title="Failed to load Munki software"
              error={software.error}
              onRetry={() => void software.refetch()}
            />
          ) : software.isLoading ? null : software.data?.count === 0 ? (
            <EmptyPanel>No software in this host&apos;s manifest</EmptyPanel>
          ) : (
            <DataTableStatic columns={softwareColumns} data={software.data?.items ?? []} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function MunkiSoftwareCell({ software }: { software: MunkiHostManifestSoftware }) {
  const label =
    software.package.strategy === "specific"
      ? `${software.software.name} (${software.package.version})`
      : software.software.name;

  return (
    <Link
      to="/munki/software/$id"
      params={{ id: String(software.software.id) }}
      className="flex min-w-0 items-center gap-2 font-medium hover:underline"
    >
      <SoftwareArtwork src={software.software.icon_url} />
      <span className="truncate">{label}</span>
    </Link>
  );
}

function MunkiActionBadge({ action }: { action: MunkiSoftwareAction }) {
  const presentation = MUNKI_SOFTWARE_ACTIONS[action];
  const Icon = actionIcons[action];
  const badge = (
    <Badge variant="secondary">
      <Icon data-icon="inline-start" />
      {presentation.name}
    </Badge>
  );

  return (
    <Tooltip>
      <TooltipTrigger render={badge} />
      <TooltipContent className="max-w-72 text-left">{presentation.description}</TooltipContent>
    </Tooltip>
  );
}

function MunkiSoftwareStatusBadge({ software }: { software: MunkiHostManifestSoftware }) {
  const status = munkiSoftwareStatus(software);
  const Icon = status.icon;
  const badge = (
    <Badge variant={status.variant}>
      <Icon data-icon="inline-start" />
      {status.label}
    </Badge>
  );

  return (
    <Tooltip>
      <TooltipTrigger render={badge} />
      <TooltipContent className="max-w-72 text-left">{status.description}</TooltipContent>
    </Tooltip>
  );
}

interface StatusPresentation {
  label: string;
  description: string;
  icon: LucideIcon;
  variant: "outline" | "secondary";
}

function munkiSoftwareStatus(software: MunkiHostManifestSoftware): StatusPresentation {
  const observation = software.observation;
  if (!observation) {
    return {
      label: "No report",
      description: "No exact-name result was present in the latest Munki software report.",
      icon: Clock3,
      variant: "outline",
    };
  }

  if (observation.installed) {
    return {
      label: observation.installed_version
        ? `Installed ${observation.installed_version}`
        : "Installed",
      description: observation.installed_version
        ? `Munki reported version ${observation.installed_version} as installed.`
        : "Munki reported its required version or a newer version as installed without a version value.",
      icon: CircleCheck,
      variant: "outline",
    };
  }

  if (
    software.actions.includes("managed_updates") &&
    software.package.strategy === "specific" &&
    observation.target_version === software.package.version
  ) {
    return {
      label: "Update pending",
      description: "Munki reported that the currently pinned update version is not installed.",
      icon: RefreshCw,
      variant: "secondary",
    };
  }

  if (software.actions.includes("managed_installs")) {
    return {
      label: "Pending",
      description:
        "Munki reported that its required version was not installed; this does not distinguish missing from outdated software.",
      icon: Download,
      variant: "secondary",
    };
  }

  if (
    software.actions.includes("optional_installs") ||
    software.actions.includes("featured_items") ||
    software.actions.includes("default_installs")
  ) {
    return {
      label: "Available",
      description:
        "Munki reported that the offered version was not installed; an older version may still be present.",
      icon: PackageOpen,
      variant: "outline",
    };
  }

  return {
    label: "Pending",
    description:
      "Munki reported that its required version was not installed; the report does not prove the current desired action has completed.",
    icon: Clock3,
    variant: "secondary",
  };
}

function MunkiStatusBadge({ munki }: { munki: MunkiHostState }) {
  if (munki.errors.length > 0) {
    return <Badge variant="destructive">Failed</Badge>;
  }
  if (munki.problem_installs.length > 0) {
    return <Badge variant="secondary">Problems</Badge>;
  }
  if (munki.warnings.length > 0) {
    return <Badge variant="secondary">Warnings</Badge>;
  }
  return <Badge variant="outline">OK</Badge>;
}

function problemRows(kind: string, values: string[]) {
  return values.map((value) => ({ kind, value }));
}
