import { getRouteApi, useParams } from "@tanstack/react-router";
import type { ColumnDef, Table } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { LabelTargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import { Skeleton } from "@components/ui/skeleton";
import { TabsContent, TabsTrigger } from "@components/ui/tabs";
import { useAuth } from "@features/auth/queries";
import {
  CHECK_RESULT_STATUSES,
  CHECK_RESULT_STATUS_OPTIONS,
  parseCheckResultStatus,
} from "@features/osquery/checks/model";
import { LiveRunButton, ShowQueryButton } from "@features/osquery/live/query-actions";
import type { OsqueryCheckHostStatus } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { CheckDeleteDialog } from "./delete-dialog";
import { listAllCheckResults, useCheck, useCheckResults } from "./queries";

const resultColumns: ColumnDef<OsqueryCheckHostStatus>[] = [
  {
    accessorKey: "host_name",
    header: () => "Host",
    cell: ({ row }) => (
      <Link to="/hosts/$id" params={{ id: String(row.original.host_id) }} className="font-medium">
        {row.original.host_name}
      </Link>
    ),
  },
  {
    accessorKey: "status",
    header: () => "Status",
    enableColumnFilter: true,
    cell: ({ row }) => (
      <EnumStatusIndicator value={row.original.status} metadata={CHECK_RESULT_STATUSES} />
    ),
  },
  {
    accessorKey: "updated_at",
    header: () => "Last Evaluated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
  },
];

const resultExportColumns: DataTableExportOptions<OsqueryCheckHostStatus>["columns"] = [
  { header: "Host", value: (row) => row.host_name },
  { header: "Status", value: (row) => row.status },
  { header: "Last Evaluated", value: (row) => row.updated_at },
];

const STATUS_FILTER_KEYS = [{ id: "status" }] as const;
const routeApi = getRouteApi("/_authenticated/osquery/checks/$id/");

function CheckResultsToolbar({ table }: { table: Table<OsqueryCheckHostStatus> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={CHECK_RESULT_STATUS_OPTIONS}
    />
  );
}

export function CheckDetailPage() {
  const { id: checkId } = useParams({
    from: "/_authenticated/osquery/checks/$id",
  });
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const activeTab = search.tab === "results" ? "results" : "overview";
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleteOpen, setDeleteOpen] = useState(false);
  const id = parseRouteID(checkId);
  const check = useCheck(id);
  const status = parseCheckResultStatus(tableSearch.filters.status?.[0]);
  const results = useCheckResults(activeTab === "results" ? id : null, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
  });
  const rows = results.data?.items ?? [];
  const totalCount = results.data?.count ?? 0;
  const pageCount = results.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: resultColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => String(row.host_id),
  });

  if (id === null) {
    return (
      <QueryGate title="Failed to load check" error={{ message: "Check route is invalid." }} />
    );
  }

  if (check.error) {
    return (
      <QueryGate
        title="Failed to load check"
        error={check.error}
        onRetry={() => void check.refetch()}
      />
    );
  }

  if (!check.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  const exportOptions: DataTableExportOptions<OsqueryCheckHostStatus> = {
    filename: `osquery-check-${id}-results`,
    columns: resultExportColumns,
    loadRows: () =>
      listAllCheckResults(id, {
        q: tableSearch.q,
        sort: tableSearch.sort,
        status,
      }),
  };
  return (
    <PageShell>
      <PageHeader
        title="Check Details"
        meta={`Edited ${formatRelative(check.data.updated_at)}`}
        actions={
          <>
            {isAdmin ? (
              <>
                <Button
                  size="sm"
                  render={<Link to="/osquery/checks/$id/edit" params={{ id: checkId }} />}
                  nativeButton={false}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => setDeleteOpen(true)}
                >
                  <Trash2 data-icon="inline-start" />
                  Delete
                </Button>
              </>
            ) : null}
            <ShowQueryButton sql={check.data.query} />
            <LiveRunButton kind="check" id={id} sql={check.data.query} />
          </>
        }
      />

      <ScrollableTabs value={activeTab}>
        <ScrollableTabsList>
          <TabsTrigger
            value="overview"
            render={
              <Link
                to="/osquery/checks/$id"
                params={{ id: checkId }}
                search={{ ...search, tab: undefined }}
              />
            }
            nativeButton={false}
          >
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="results"
            render={
              <Link
                to="/osquery/checks/$id"
                params={{ id: checkId }}
                search={{ ...search, tab: "results" }}
              />
            }
            nativeButton={false}
          >
            Results
          </TabsTrigger>
        </ScrollableTabsList>

        <TabsContent value="overview" className="flex flex-col gap-5">
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={check.data.name} />
            <KeyValueRow label="Description" value={check.data.description} />
            <KeyValueRow label="Passing" value={formatHostCount(check.data.passing_host_count)} />
            <KeyValueRow label="Failing" value={formatHostCount(check.data.failing_host_count)} />
          </KeyValueSection>

          <LabelTargetDetails targets={check.data.targets} />
        </TabsContent>

        <TabsContent value="results">
          {results.error ? (
            <QueryError
              title="Failed to load check results"
              error={results.error}
              onRetry={() => void results.refetch()}
            />
          ) : results.isLoading ? (
            <DataTableSkeleton columnCount={3} filterCount={1} withExport withViewOptions={false} />
          ) : (
            <DataTable
              table={table}
              exportOptions={exportOptions}
              empty={
                <PanelEmptyState>
                  {tableSearch.isFiltered ? "No matching check results" : "No check results yet"}
                </PanelEmptyState>
              }
            >
              <div className="flex flex-wrap items-center gap-2">
                <DataTableSearchInput
                  value={tableSearch.q ?? ""}
                  onValueChange={tableSearch.onQueryChange}
                  placeholder="Search check results"
                  className="h-8 w-full sm:w-64"
                />
                <CheckResultsToolbar table={table} />
              </div>
            </DataTable>
          )}
        </TabsContent>
      </ScrollableTabs>

      {isAdmin ? (
        <CheckDeleteDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          check={check.data}
          onDeleted={() => void navigate({ to: "/osquery/checks" })}
        />
      ) : null}
    </PageShell>
  );
}

function formatHostCount(count: number) {
  return `${count} ${count === 1 ? "host" : "hosts"}`;
}
