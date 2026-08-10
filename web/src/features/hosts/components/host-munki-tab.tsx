import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { RelativeTime } from "@components/relative-time";
import { Badge } from "@components/ui/badge";
import { Separator } from "@components/ui/separator";
import { useHostMunkiSoftware } from "@features/hosts/queries";
import { MUNKI_SOFTWARE_ACTIONS, type MunkiSoftwareAction } from "@features/munki/software/actions";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { ApiError, MunkiHostManifestSoftware, MunkiHostState } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";
import { MAX_PAGE_SIZE } from "@lib/pagination";

type MunkiSoftwareStatus = "installed" | "pending";

const MUNKI_SOFTWARE_STATUSES = {
  installed: { name: "Installed", variant: "success" },
  pending: { name: "Pending", variant: "default" },
} satisfies StatusMetadataMap<MunkiSoftwareStatus>;

const softwareColumns: DataTableColumnDef<MunkiHostManifestSoftware>[] = [
  {
    id: "software",
    accessorFn: (row) => row.software.name,
    header: () => "Software",
    cell: ({ row }) => <MunkiSoftwareCell software={row.original} />,
    size: 240,
    minSize: 180,
  },
  {
    accessorKey: "actions",
    header: () => "Actions",
    cell: ({ row }) => (
      <div className="flex flex-nowrap gap-1">
        {row.original.actions.map((action) => (
          <MunkiActionBadge key={action} action={action} />
        ))}
      </div>
    ),
    size: 280,
    minSize: 280,
    maxSize: 280,
    enableResizing: false,
  },
  {
    id: "target_version",
    header: () => "Target Version",
    cell: ({ row }) => row.original.observation?.target_version ?? "-",
    size: 136,
    minSize: 136,
    maxSize: 136,
    enableResizing: false,
  },
  {
    id: "status",
    header: () => "Status",
    cell: ({ row }) => <MunkiSoftwareStatus software={row.original} />,
    size: 200,
    minSize: 200,
    maxSize: 200,
    enableResizing: false,
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
  hostSerial: string | undefined;
  munki: MunkiHostState | null | undefined;
  stateError: ApiError | null;
  onStateRetry: () => void;
}

export function HostMunkiTab({
  hostId,
  hostSerial,
  munki,
  stateError,
  onStateRetry,
}: HostMunkiTabProps) {
  const software = useHostMunkiSoftware(hostId, { per_page: MAX_PAGE_SIZE });
  const hasReport = Boolean(munki && (munki.run_at || munki.version || munki.manifest_name));
  // This is good enough for now: a serial-named manifest means Munki is pointed at Woodstar.
  const showManifest = Boolean(munki?.manifest_name && munki.manifest_name !== hostSerial);
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
      ) : munki && hasReport ? (
        <KeyValueSection title="Overview">
          {munki.version ? <KeyValueRow label="Version" value={munki.version} /> : null}
          {showManifest ? <KeyValueRow label="Manifest" value={munki.manifest_name} /> : null}
          {munki.run_at ? (
            <KeyValueRow label="Last Run" value={<RelativeTime value={munki.run_at} />} />
          ) : null}
        </KeyValueSection>
      ) : (
        <PanelEmptyState>No Munki report</PanelEmptyState>
      )}

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
      <TextLink
        to="/munki/software/$id"
        params={{ id: String(software.software.id) }}
        className="min-w-0 truncate font-medium"
        title={label}
      >
        {label}
      </TextLink>
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

function MunkiSoftwareStatus({ software }: { software: MunkiHostManifestSoftware }) {
  const status = munkiSoftwareStatus(software);
  if (status === null) return "-";
  return <EnumStatusIndicator value={status} metadata={MUNKI_SOFTWARE_STATUSES} />;
}

function munkiSoftwareStatus(software: MunkiHostManifestSoftware): MunkiSoftwareStatus | null {
  const observation = software.observation;
  if (!observation) return null;

  if (observation.target_version) {
    return "pending";
  }

  if (observation.installed) {
    return "installed";
  }

  return null;
}

function problemRows(kind: string, values: string[]): ProblemRow[] {
  return values.map((value) => ({ kind, value }));
}
