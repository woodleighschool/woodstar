import { Link } from "@tanstack/react-router";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { Activity, FolderLock, ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState } from "@/components/data-table";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useHostSantaRules, type HostDetail, type HostSantaRule } from "@/hooks/use-hosts";
import { tableQueryParams } from "@/hooks/use-table-pagination-params";
import { clientModeLabel } from "@/lib/santa-configurations";
import { policyLabel, ruleTypeLabel } from "@/lib/santa-rules";
import { formatRelative } from "@/lib/utils";

const HOST_SANTA_RULES_PAGE_SIZE = 25;

export function HostSantaTab({ hostId, host }: { hostId: number | null; host: HostDetail }) {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: HOST_SANTA_RULES_PAGE_SIZE,
  });
  const [sorting, setSorting] = useState<SortingState>([]);
  const rules = useHostSantaRules(hostId, tableQueryParams({ pagination, sorting }));
  const santa = host.santa;
  const items = rules.data?.items ?? [];
  const totalCount = rules.data?.count ?? 0;
  const matchedLabelName = santa?.configuration?.matched_via_label?.name;

  const configurationValue = santa?.configuration?.name ? (
    matchedLabelName ? (
      <Tooltip>
        <TooltipTrigger asChild>
          <span>{santa.configuration.name}</span>
        </TooltipTrigger>
        <TooltipContent>
          <div>{`Matched via label: ${matchedLabelName}`}</div>
        </TooltipContent>
      </Tooltip>
    ) : (
      santa.configuration.name
    )
  ) : (
    "-"
  );

  const columns = useMemo<ColumnDef<HostSantaRule>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
        cell: ({ row }) => row.original.name || "-",
      },
      {
        accessorKey: "identifier",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Identifier" />,
        cell: ({ row }) => row.original.identifier,
      },
      {
        accessorKey: "rule_type",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
        cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
      },
      {
        accessorKey: "policy",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Policy" />,
        cell: ({ row }) => policyLabel(row.original.policy),
      },
      {
        accessorKey: "applied",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
        cell: ({ row }) => (
          <Badge variant={!row.original.applied ? "secondary" : "outline"} className="gap-1.5">
            <span
              className={
                !row.original.applied ? "bg-warning size-1.5 rounded-full" : "bg-status-online size-1.5 rounded-full"
              }
            />
            {row.original.applied ? "Applied" : "Pending"}
          </Badge>
        ),
      },
    ],
    [],
  );

  if (!santa) return null;

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent>
          <div className="mb-5 flex flex-wrap gap-2">
            <Button asChild variant="outline" size="sm">
              <Link to="/santa/events" search={{ host_id: host.id }}>
                <Activity data-icon="inline-start" />
                View Execution Events
              </Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link to="/santa/events/file-access" search={{ host_id: host.id }}>
                <FolderLock data-icon="inline-start" />
                View File Access Events
              </Link>
            </Button>
          </div>
          <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
            {[
              { label: "Version", value: santa.version || "-" },
              { label: "Client Mode", value: clientModeLabel(santa.client_mode_reported) },
              { label: "Configuration", value: configurationValue },
              { label: "Last Sync", value: formatRelative(santa.last_sync_at) },
              {
                label: "Rule Sync",
                value: `${santa.rule_sync.applied_count} applied / ${santa.rule_sync.desired_count} desired`,
              },
              { label: "Pending Rules", value: String(santa.rule_sync.pending_count) },
            ].map((tile) => (
              <div key={tile.label} className="flex min-w-0 flex-col gap-1">
                <dt className="text-muted-foreground text-xs font-semibold">{tile.label}</dt>
                <dd className="text-foreground truncate text-sm">{tile.value}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      <Card className="gap-4 py-4">
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <CardTitle>Rules</CardTitle>
        </CardHeader>
        <CardContent>
          {rules.error ? (
            <Alert variant="destructive">
              <AlertTitle>Failed to Load Rules</AlertTitle>
              <AlertDescription>{rules.error.message}</AlertDescription>
            </Alert>
          ) : (
            <DataTable
              columns={columns}
              data={items}
              totalCount={totalCount}
              pagination={pagination}
              sorting={sorting}
              onPaginationChange={setPagination}
              onSortingChange={setSorting}
              isLoading={rules.isLoading}
              getRowId={(rule) => `${rule.rule_id}-${rule.matched_include_id}`}
              rowHref={(rule) => ({
                to: "/santa/rules/$ruleId",
                params: { ruleId: String(rule.rule_id) },
              })}
              empty={
                <DataTableEmptyState
                  icon={<ShieldCheck />}
                  title="No Rules"
                  description="No Santa rules match this host."
                />
              }
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
