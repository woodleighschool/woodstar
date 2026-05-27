import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useSantaEvents, type SantaEvent } from "@/hooks/use-santa";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";

const DECISION_FILTERS = [
  { value: "allowed", label: "Allowed" },
  { value: "blocked", label: "Blocked" },
  { value: "allow_unknown", label: "Allow unknown" },
  { value: "allow_binary", label: "Allow binary" },
  { value: "allow_certificate", label: "Allow certificate" },
  { value: "allow_scope", label: "Allow scope" },
  { value: "allow_teamid", label: "Allow Team ID" },
  { value: "allow_signingid", label: "Allow signing ID" },
  { value: "allow_cdhash", label: "Allow CDHash" },
  { value: "block_unknown", label: "Block unknown" },
  { value: "block_binary", label: "Block binary" },
  { value: "block_certificate", label: "Block certificate" },
  { value: "block_scope", label: "Block scope" },
  { value: "block_teamid", label: "Block Team ID" },
  { value: "block_signingid", label: "Block signing ID" },
  { value: "block_cdhash", label: "Block CDHash" },
  { value: "bundle_binary", label: "Bundle binary" },
  { value: "unknown", label: "Unknown" },
] as const;

export function SantaEventsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const decisions = search.decisions ?? [];
  const query = useSantaEvents({
    q: search.q,
    decisions,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || decisions.length > 0;

  const columns: ColumnDef<SantaEvent>[] = [
    {
      id: "file_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Executable" />,
      cell: ({ row }) => (
        <div className="grid gap-1">
          <span className="font-medium">{row.original.executable.file_name || row.original.executable.sha256}</span>
          <span className="text-muted-foreground max-w-[34rem] truncate text-xs">
            {row.original.file_path || row.original.executable.sha256}
          </span>
        </div>
      ),
    },
    {
      id: "decision",
      accessorKey: "decision",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Decision" />,
      cell: ({ row }) => <DecisionBadge decision={row.original.decision} />,
    },
    {
      id: "host_id",
      accessorKey: "host_id",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => (
        <Link
          to="/hosts/$hostId"
          params={{ hostId: String(row.original.host_id) }}
          className="font-medium hover:underline"
        >
          {row.original.host_id}
        </Link>
      ),
    },
    {
      id: "executing_user",
      accessorKey: "executing_user",
      header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.executing_user || "-"}</span>,
    },
    {
      id: "occurred_at",
      accessorFn: (row) => row.occurred_at ?? row.ingested_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="When" />,
      cell: ({ row }) => {
        const timestamp = row.original.occurred_at ?? row.original.ingested_at;
        return (
          <span className="text-muted-foreground" title={new Date(timestamp).toLocaleString()}>
            {formatRelative(timestamp)}
          </span>
        );
      },
    },
  ];

  return (
    <PageShell>
      <PageHeader title="Santa events" description="Recent execution events uploaded by Santa clients." />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load Santa events</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          toolbar={
            <div className="flex flex-wrap items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" label="Search Santa events" />
              <DataTableFacetedFilter
                title="Decision"
                options={[...DECISION_FILTERS]}
                selected={decisions}
                onChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              />
            </div>
          }
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No matches" : "No execution events"}
              description={
                hasFilters
                  ? "No Santa execution events matched these filters."
                  : "Client allow/block decisions appear here after Santa uploads them."
              }
            />
          }
        />
      )}
    </PageShell>
  );
}

function DecisionBadge({ decision }: { decision: string }) {
  const blocked = decision.startsWith("block_");
  const allowed = decision.startsWith("allow_");
  return (
    <Badge variant={blocked ? "destructive" : allowed ? "secondary" : "outline"}>{decision.replaceAll("_", " ")}</Badge>
  );
}
