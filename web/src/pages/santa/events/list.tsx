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
import { Button } from "@/components/ui/button";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useSantaEvents,
  useSantaFileAccessEvents,
  type SantaEvent,
  type SantaFileAccessEvent,
} from "@/hooks/use-santa";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { DECISION_FILTERS, FILE_ACCESS_DECISION_FILTERS, fileName } from "./constants";
import { DecisionBadge, HostLink, Timestamp } from "./event-ui";

type EventListKind = "execution" | "file-access";

export function SantaEventsPage() {
  return (
    <PageShell>
      <PageHeader title="Events" description="Review Santa execution and file access activity." />
      <EventListNav active="execution" />
      <ExecutionEventsTable />
    </PageShell>
  );
}

export function SantaFileAccessEventsPage() {
  return (
    <PageShell>
      <PageHeader title="Events" description="Review Santa execution and file access activity." />
      <EventListNav active="file-access" />
      <FileAccessEventsTable />
    </PageShell>
  );
}

function EventListNav({ active }: { active: EventListKind }) {
  return (
    <div className="flex items-center gap-2">
      <Button asChild size="sm" variant={active === "execution" ? "secondary" : "ghost"}>
        <Link to="/santa/events">Execution</Link>
      </Button>
      <Button asChild size="sm" variant={active === "file-access" ? "secondary" : "ghost"}>
        <Link to="/santa/events/file-access">File Access</Link>
      </Button>
    </div>
  );
}

function ExecutionEventsTable() {
  const search = useSearch({ from: "/_authenticated/santa/events/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
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
          <span className="max-w-[34rem] truncate text-xs">
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
      accessorFn: (row) => row.host.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => <HostLink host={row.original.host} />,
    },
    {
      id: "executing_user",
      accessorKey: "executing_user",
      header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
      cell: ({ row }) => row.original.executing_user || "-",
    },
    {
      id: "occurred_at",
      accessorFn: (row) => row.occurred_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="When" />,
      cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
    },
  ];

  return (
    <>
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Execution Events</AlertTitle>
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
          rowHref={(row) => ({ to: "/santa/events/$eventId", params: { eventId: String(row.id) } })}
          toolbar={
            <EventTableToolbar
              draft={draft}
              setDraft={setDraft}
              decisions={decisions}
              decisionOptions={[...DECISION_FILTERS]}
              onDecisionsChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              searchPlaceholder="Search"
            />
          }
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No Matches" : "No Execution Events"}
              description={
                hasFilters ? "No events matched these filters." : "Client decisions appear after Santa syncs."
              }
            />
          }
        />
      )}
    </>
  );
}

function FileAccessEventsTable() {
  const search = useSearch({ from: "/_authenticated/santa/events/file-access/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
  const decisions = search.decisions ?? [];
  const query = useSantaFileAccessEvents({
    q: search.q,
    decisions,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || decisions.length > 0;

  const columns: ColumnDef<SantaFileAccessEvent>[] = [
    {
      id: "target",
      accessorKey: "target",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Target" />,
      cell: ({ row }) => (
        <span className="block max-w-96 truncate font-medium" title={row.original.target}>
          {fileName(row.original.target) || row.original.target}
        </span>
      ),
    },
    {
      id: "decision",
      accessorKey: "decision",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Decision" />,
      cell: ({ row }) => <DecisionBadge decision={row.original.decision} />,
    },
    {
      id: "host",
      accessorFn: (row) => row.host.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => <HostLink host={row.original.host} />,
    },
    {
      id: "process",
      header: "Process",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="block max-w-60 truncate">
          {row.original.primary_process.file_name || fileName(row.original.primary_process.file_path) || "-"}
        </span>
      ),
    },
    {
      id: "rule_name",
      accessorKey: "rule_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Rule" />,
      cell: ({ row }) => <span>{row.original.rule_name || row.original.rule_version || "-"}</span>,
    },
    {
      id: "occurred_at",
      accessorFn: (row) => row.occurred_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Occurred" />,
      cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
    },
  ];

  return (
    <>
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load File Access Events</AlertTitle>
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
          rowHref={(row) => ({ to: "/santa/events/file-access/$eventId", params: { eventId: String(row.id) } })}
          toolbar={
            <EventTableToolbar
              draft={draft}
              setDraft={setDraft}
              decisions={decisions}
              decisionOptions={[...FILE_ACCESS_DECISION_FILTERS]}
              onDecisionsChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              searchPlaceholder="Search Target, Process, Host, Signer"
            />
          }
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No Matches" : "No File Access Events"}
              description={
                hasFilters
                  ? "No file access events matched these filters."
                  : "File access decisions appear after Santa syncs."
              }
            />
          }
        />
      )}
    </>
  );
}

function EventTableToolbar({
  draft,
  setDraft,
  decisions,
  decisionOptions,
  onDecisionsChange,
  searchPlaceholder,
}: {
  draft: string;
  setDraft: (next: string) => void;
  decisions: string[];
  decisionOptions: Array<{ value: string; label: string }>;
  onDecisionsChange: (next: string[]) => void;
  searchPlaceholder: string;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={setDraft} placeholder={searchPlaceholder} />
      <DataTableFacetedFilter
        title="Decision"
        options={decisionOptions}
        selected={decisions}
        onChange={onDecisionsChange}
      />
    </div>
  );
}
