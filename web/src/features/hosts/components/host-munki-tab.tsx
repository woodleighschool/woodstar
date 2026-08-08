import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@components/ui/hover-card";
import { Separator } from "@components/ui/separator";
import { useHostMunkiSoftware } from "@features/hosts/queries";
import { MunkiActionBadges } from "@features/munki/software/action-badges";
import {
  InstallationStatusText,
  LastCollected,
  MunkiResultText,
} from "@features/munki/software/deployment";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { ApiError, MunkiHostManifestSoftware, MunkiHostState } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";
import { formatDateTime, formatRelative } from "@lib/utils";

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
    cell: ({ row }) => <MunkiActionBadges actions={row.original.actions} />,
  },
  {
    id: "status",
    accessorKey: "status",
    header: () => "Status",
    cell: ({ row }) => <InstallationStatusText status={row.original.status} />,
  },
  {
    id: "installed_version",
    accessorKey: "installed_version",
    header: () => "Installed version",
    cell: ({ row }) => row.original.installed_version || "-",
  },
  {
    id: "munki_result",
    accessorKey: "munki_result",
    header: () => "Munki result",
    cell: ({ row }) => <MunkiResultText result={row.original.munki_result} />,
  },
  {
    id: "target_version",
    accessorKey: "target_version",
    header: () => "Target",
    cell: ({ row }) => row.original.target_version || "-",
  },
  {
    id: "last_collected_at",
    accessorKey: "last_collected_at",
    header: () => "Last Collected",
    cell: ({ row }) => <LastCollected value={row.original.last_collected_at} />,
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
        ...problemRows("Collection Error", munki.collection_error ? [munki.collection_error] : []),
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
        <PanelEmptyState>No Munki report collected</PanelEmptyState>
      ) : munki ? (
        <KeyValueSection title="Overview">
          <KeyValueRow label="Version" value={munki.version} />
          <KeyValueRow label="Manifest" value={munki.manifest_name} />
          <KeyValueRow label="Report" value={<MunkiReportState munki={munki} />} />
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
  const label = software.software.name;

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

function MunkiReportBadge({ state }: { state: string }) {
  switch (state) {
    case "current":
      return <Badge variant="success">Current</Badge>;
    case "no_report":
      return <Badge variant="warning">No report</Badge>;
    case "collection_failed":
      return <Badge variant="error">Collection failed</Badge>;
    case "never_collected":
      return <Badge variant="outline">Never collected</Badge>;
    default:
      return <Badge variant="outline">{state}</Badge>;
  }
}

function MunkiReportState({ munki }: { munki: MunkiHostState }) {
  return (
    <HoverCard>
      <HoverCardTrigger
        render={<Button type="button" variant="ghost" size="xs" className="h-auto p-0" />}
      >
        <MunkiReportBadge state={munki.report_state} />
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-80">
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-xs">
          <dt className="text-muted-foreground">Last attempt</dt>
          <dd>
            <EvidenceTimestamp value={munki.last_attempt_at} />
          </dd>
          <dt className="text-muted-foreground">Last successful</dt>
          <dd>
            <EvidenceTimestamp value={munki.last_successful_at} />
          </dd>
          <dt className="text-muted-foreground">Run started</dt>
          <dd>
            <EvidenceTimestamp value={munki.run_started_at} />
          </dd>
          <dt className="text-muted-foreground">Run ended</dt>
          <dd>
            <EvidenceTimestamp value={munki.run_ended_at} />
          </dd>
        </dl>
      </HoverCardContent>
    </HoverCard>
  );
}

function EvidenceTimestamp({ value }: { value: string | undefined }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return <span title={formatDateTime(value)}>{formatRelative(value)}</span>;
}

function problemRows(kind: string, values: string[]): ProblemRow[] {
  return values.map((value) => ({ kind, value }));
}
