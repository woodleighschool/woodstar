import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useHostSantaEffectiveRules, type HostDetail, type HostSantaEffectiveRule } from "@/hooks/use-hosts";
import { tableQueryParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";

const HOST_SANTA_RULES_PAGE_SIZE = 25;

export function HostSantaTab({ hostId, host }: { hostId: number | null; host: HostDetail }) {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: HOST_SANTA_RULES_PAGE_SIZE,
  });
  const [sorting, setSorting] = useState<SortingState>([]);
  const rules = useHostSantaEffectiveRules(hostId, tableQueryParams({ pagination, sorting }));
  const santa = host.santa;
  const items = rules.data?.items ?? [];
  const totalCount = rules.data?.count ?? 0;

  const columns = useMemo<ColumnDef<HostSantaEffectiveRule>[]>(
    () => [
      {
        accessorKey: "identifier",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Rule" />,
        cell: ({ row }) => (
          <div className="grid gap-1">
            <span className="font-medium">{row.original.identifier}</span>
            <span className="text-muted-foreground text-xs">{row.original.rule_type}</span>
          </div>
        ),
      },
      {
        accessorKey: "policy",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Policy" />,
        cell: ({ row }) => row.original.policy,
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
          <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
            {[
              { label: "Version", value: santa.version || "-" },
              { label: "Reported mode", value: santa.client_mode_reported },
              { label: "Effective configuration", value: santa.effective_configuration?.name ?? "-" },
              { label: "Matched label", value: santa.effective_configuration?.matched_via_label?.name ?? "-" },
              { label: "Last sync", value: formatRelative(santa.last_sync_at) },
              {
                label: "Rule sync",
                value: `${santa.rule_sync.applied_count} applied / ${santa.rule_sync.desired_count} desired`,
              },
              { label: "Pending rules", value: String(santa.rule_sync.pending_count) },
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
          <CardTitle>Effective rules</CardTitle>
          {items.length > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {items.length} {items.length === 1 ? "rule" : "rules"}
            </span>
          ) : null}
        </CardHeader>
        <CardContent>
          {rules.error ? (
            <Alert variant="destructive">
              <AlertTitle>Failed to load effective rules</AlertTitle>
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
              empty={
                <DataTableEmptyState
                  icon={<ShieldCheck />}
                  title="No effective rules"
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
