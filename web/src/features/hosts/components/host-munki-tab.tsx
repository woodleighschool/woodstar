import {
  CircleCheck,
  Clock3,
  Download,
  PackageOpen,
  RefreshCw,
  type LucideIcon,
} from "lucide-react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Badge } from "@components/ui/badge";
import { Separator } from "@components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import { useHostMunkiSoftware } from "@features/hosts/queries";
import { MUNKI_SOFTWARE_ACTIONS, type MunkiSoftwareAction } from "@features/munki/software/actions";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { ApiError, MunkiHostManifestSoftware, MunkiHostState } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

const softwareColumns: DataTableColumnDef<MunkiHostManifestSoftware>[] = [
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

interface ProblemRow {
  kind: string;
  value: string;
}

const problemColumns: DataTableColumnDef<ProblemRow>[] = [
  {
    accessorKey: "kind",
    header: () => "Type",
    cell: ({ row }) => <Badge variant="secondary">{row.original.kind}</Badge>,
  },
  {
    accessorKey: "value",
    header: () => "Message",
    cell: ({ row }) => <span className="wrap-break-word">{row.original.value}</span>,
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
        <PanelEmptyState>No Munki run reported</PanelEmptyState>
      ) : munki ? (
        <KeyValueSection title="Overview">
          <KeyValueRow label="Version" value={munki.version} />
          <KeyValueRow label="Manifest" value={munki.manifest_name} />
          <KeyValueRow label="Status" value={<MunkiStatusBadge munki={munki} />} />
          <KeyValueRow label="Last Run Started" value={formatRelative(munki.run_started_at)} />
          <KeyValueRow label="Last Run Ended" value={formatRelative(munki.run_ended_at)} />
        </KeyValueSection>
      ) : null}

      {problems.length > 0 ? (
        <DataTableStatic heading="Problems" columns={problemColumns} data={problems} />
      ) : null}

      {software.error ? (
        <section className="flex min-w-0 flex-col gap-3">
          <h2 className="text-base/snug font-medium text-foreground">Software</h2>
          <Separator />
          <div className="px-3">
            <QueryError
              title="Failed to load Munki software"
              error={software.error}
              onRetry={() => void software.refetch()}
            />
          </div>
        </section>
      ) : software.isLoading ? null : (
        <DataTableStatic
          heading="Software"
          columns={softwareColumns}
          data={software.data?.items ?? []}
          empty={<PanelEmptyState>No software in this host&apos;s manifest</PanelEmptyState>}
        />
      )}
    </div>
  );
}

function MunkiSoftwareCell({ software }: { software: MunkiHostManifestSoftware }) {
  const label =
    software.package.strategy === "specific"
      ? `${software.software.name} (${software.package.version})`
      : software.software.name;

  return (
    <div className="flex min-w-0 items-center gap-2">
      <SoftwareArtwork src={software.software.icon_url} />
      <Link
        to="/munki/software/$id"
        params={{ id: String(software.software.id) }}
        className="min-w-0 truncate font-medium"
        title={label}
      >
        {label}
      </Link>
    </div>
  );
}

function MunkiActionBadge({ action }: { action: MunkiSoftwareAction }) {
  return (
    <Badge variant="secondary" className="font-normal">
      {MUNKI_SOFTWARE_ACTIONS[action].name}
    </Badge>
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

function problemRows(kind: string, values: string[]): ProblemRow[] {
  return values.map((value) => ({ kind, value }));
}
