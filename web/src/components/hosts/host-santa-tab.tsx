import type { ColumnDef } from "@tanstack/react-table";
import { Activity, FolderLock } from "lucide-react";
import { useMemo } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { EnumBadge } from "@/components/enum-badge";
import { EnumStatus } from "@/components/enum-status";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { Link } from "@/components/link";
import { PathText } from "@/components/path-text";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/hooks/use-auth";
import { useHostSantaRules } from "@/hooks/use-hosts";
import type { ApiError, SantaHostState, SantaRuleStatus } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { CLIENT_MODES } from "@/lib/santa-configurations";
import { POLICIES, RULE_TYPES } from "@/lib/santa-rules";
import { formatRelative } from "@/lib/utils";

const RULE_APPLICATION_STATUSES = {
  applied: { name: "Applied", variant: "success" },
  pending: { name: "Pending", variant: "warning" },
} as const;

function santaRuleColumns(isAdmin: boolean): ColumnDef<SantaRuleStatus>[] {
  return [
    {
      accessorKey: "name",
      header: () => "Name",
      cell: ({ row }) =>
        isAdmin ? (
          <Link
            to="/santa/rules/$id"
            params={{ id: String(row.original.rule_id) }}
            className="font-medium"
          >
            {row.original.name}
          </Link>
        ) : (
          row.original.name
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
        <EnumStatus
          value={row.original.applied ? "applied" : "pending"}
          metadata={RULE_APPLICATION_STATUSES}
        />
      ),
    },
  ];
}

interface HostSantaTabProps {
  hostId: number;
  santa: SantaHostState | null | undefined;
  stateError: ApiError | null;
  onStateRetry: () => void;
}

export function HostSantaTab({ hostId, santa, stateError, onStateRetry }: HostSantaTabProps) {
  const { user } = useAuth();
  const columns = useMemo(() => santaRuleColumns(user?.role === "admin"), [user?.role]);
  const rules = useHostSantaRules(hostId, { per_page: MAX_PAGE_SIZE });
  const items = rules.data?.items ?? [];
  const totalCount = rules.data?.count ?? 0;
  const configuration = santa?.configuration;
  return (
    <div className="flex flex-col gap-4">
      {stateError ? (
        <QueryError title="Failed to load Santa state" error={stateError} onRetry={onStateRetry} />
      ) : santa === null ? (
        <EmptyPanel>No Santa sync reported</EmptyPanel>
      ) : santa ? (
        <Card>
          <CardContent>
            <div className="mb-5 flex flex-wrap gap-2">
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
            <KeyValueGrid>
              <KeyValueItem label="Version" value={santa.version} />
              <KeyValueItem
                label="Client Mode"
                value={<EnumStatus value={santa.client_mode_reported} metadata={CLIENT_MODES} />}
              />
              <KeyValueItem
                label="Configuration"
                value={
                  configuration ? <SantaConfigurationLink configuration={configuration} /> : null
                }
              />
              <KeyValueItem label="Last Sync" value={formatRelative(santa.last_seen_at)} />
              <KeyValueItem
                label="Rule Sync"
                value={`${santa.rule_sync.applied_count} applied / ${santa.rule_sync.desired_count} desired`}
              />
              <KeyValueItem label="Pending Rules" value={santa.rule_sync.pending_count} />
            </KeyValueGrid>
          </CardContent>
        </Card>
      ) : null}

      <Card className="gap-4 py-4">
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <CardTitle>Rules</CardTitle>
        </CardHeader>
        <CardContent>
          {rules.error ? (
            <QueryError
              title="Failed to load rules"
              error={rules.error}
              onRetry={() => void rules.refetch()}
            />
          ) : rules.isLoading ? null : totalCount === 0 ? (
            <EmptyPanel>No matching rules</EmptyPanel>
          ) : (
            <DataTableStatic columns={columns} data={items} />
          )}
        </CardContent>
      </Card>
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
