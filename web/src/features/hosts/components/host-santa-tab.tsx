import type { ColumnDef } from "@tanstack/react-table";
import { Activity, FolderLock } from "lucide-react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import { EnumBadge } from "@components/enum-badge";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { PathText } from "@components/path-text";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Separator } from "@components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import { useHostSantaRules } from "@features/hosts/queries";
import { CLIENT_MODES } from "@features/santa/configurations/metadata";
import { POLICIES, RULE_TYPES } from "@features/santa/rules/metadata";
import type { ApiError, SantaHostState, SantaRuleStatus } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

const RULE_APPLICATION_STATUSES = {
  applied: { name: "Applied", variant: "success" },
  pending: { name: "Pending", variant: "warning" },
} as const;

const santaRuleColumns: ColumnDef<SantaRuleStatus>[] = [
  {
    accessorKey: "name",
    header: () => "Name",
    cell: ({ row }) => (
      <Link
        to="/santa/rules/$id"
        params={{ id: String(row.original.rule_id) }}
        className="font-medium"
      >
        {row.original.name}
      </Link>
    ),
  },
  {
    accessorKey: "rule_type",
    header: () => "Rule Type",
    cell: ({ row }) => <EnumBadge value={row.original.rule_type} metadata={RULE_TYPES} />,
  },
  {
    accessorKey: "identifier",
    header: () => "Identifier",
    cell: ({ row }) => <PathText value={row.original.identifier} />,
  },
  {
    accessorKey: "policy",
    header: () => "Policy",
    cell: ({ row }) => <EnumBadge value={row.original.policy} metadata={POLICIES} />,
  },
  {
    accessorKey: "applied",
    header: () => "Status",
    cell: ({ row }) => (
      <EnumStatusIndicator
        value={row.original.applied ? "applied" : "pending"}
        metadata={RULE_APPLICATION_STATUSES}
      />
    ),
  },
];

interface HostSantaTabProps {
  hostId: number;
  santa: SantaHostState | null | undefined;
  stateError: ApiError | null;
  onStateRetry: () => void;
}

export function HostSantaTab({ hostId, santa, stateError, onStateRetry }: HostSantaTabProps) {
  const rules = useHostSantaRules(hostId, { per_page: MAX_PAGE_SIZE });
  const items = rules.data?.items ?? [];
  const configuration = santa?.configuration;
  return (
    <div className="flex flex-col gap-4">
      {stateError ? (
        <QueryError title="Failed to load Santa state" error={stateError} onRetry={onStateRetry} />
      ) : santa === null ? (
        <PanelEmptyState>No Santa sync reported</PanelEmptyState>
      ) : santa ? (
        <KeyValueSection
          title="Overview"
          actions={
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                size="sm"
                render={<Link to="/santa/events" search={{ host_id: hostId }} />}
                nativeButton={false}
              >
                <Activity data-icon="inline-start" />
                View Execution Events
              </Button>
              <Button
                variant="outline"
                size="sm"
                render={<Link to="/santa/events/file-access" search={{ host_id: hostId }} />}
                nativeButton={false}
              >
                <FolderLock data-icon="inline-start" />
                View File Access Events
              </Button>
            </div>
          }
        >
          <KeyValueRow label="Version" value={santa.version} />
          <KeyValueRow
            label="Client Mode"
            value={
              <EnumStatusIndicator value={santa.client_mode_reported} metadata={CLIENT_MODES} />
            }
          />
          <KeyValueRow
            label="Configuration"
            value={configuration ? <SantaConfigurationLink configuration={configuration} /> : null}
          />
          <KeyValueRow label="Last Sync" value={formatRelative(santa.last_seen_at)} />
          <KeyValueRow
            label="Rule Sync"
            value={`${santa.rule_sync.applied_count} applied / ${santa.rule_sync.desired_count} desired`}
          />
          <KeyValueRow label="Pending Rules" value={santa.rule_sync.pending_count} />
        </KeyValueSection>
      ) : null}

      {rules.error ? (
        <section className="flex min-w-0 flex-col gap-3">
          <h2 className="text-base/snug font-medium text-foreground">Rules</h2>
          <Separator />
          <div className="px-3">
            <QueryError
              title="Failed to load rules"
              error={rules.error}
              onRetry={() => void rules.refetch()}
            />
          </div>
        </section>
      ) : rules.isLoading ? null : (
        <DataTableStatic
          heading="Rules"
          columns={santaRuleColumns}
          data={items}
          empty={<PanelEmptyState>No matching rules</PanelEmptyState>}
        />
      )}
    </div>
  );
}

function SantaConfigurationLink({
  configuration,
}: {
  configuration: NonNullable<SantaHostState["configuration"]>;
}) {
  const link = (
    <Link
      to="/santa/configurations/$id"
      params={{ id: String(configuration.id) }}
      className="font-medium"
    >
      {configuration.name}
    </Link>
  );

  return configuration.matched_via_label ? (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent>
        <div>{`Matched via label: ${configuration.matched_via_label.name}`}</div>
      </TooltipContent>
    </Tooltip>
  ) : (
    link
  );
}
