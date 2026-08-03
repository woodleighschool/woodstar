import { getRouteApi, useParams } from "@tanstack/react-router";
import type { Table } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
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
import { CHECK_RESULT_STATUS_OPTIONS } from "@features/osquery/checks/model";
import { LiveRunButton, ShowQueryButton } from "@features/osquery/live/query-actions";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { CheckDeleteDialog } from "./delete-dialog";
import { listAllCheckResults, useCheck, useCheckResults } from "./queries";
import {
  checkResultFromStatus,
  type CheckResultRow,
  createCheckResultColumns,
} from "./query-results";

const resultColumns = createCheckResultColumns({ timestampHeader: "Last Evaluated" });

const resultExportColumns: DataTableExportOptions<CheckResultRow>["columns"] = [
  { header: "Host", value: (row) => row.host_name },
  { header: "Status", value: (row) => row.status },
  { header: "Last Evaluated", value: (row) => row.updated_at },
];

const STATUS_FILTER_KEYS = [{ id: "status", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/osquery/checks/$id/");

function CheckResultsToolbar({ table }: { table: Table<CheckResultRow> }) {
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
  const status = search.status;
  const results = useCheckResults(activeTab === "results" ? id : null, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
  });
  const rows = useMemo(
    () => results.data?.items.map(checkResultFromStatus) ?? [],
    [results.data?.items],
  );
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

  const exportOptions: DataTableExportOptions<CheckResultRow> = {
    filename: `osquery-check-${id}-results`,
    columns: resultExportColumns,
    loadRows: () =>
      listAllCheckResults(id, {
        q: tableSearch.q,
        sort: tableSearch.sort,
        status,
      }).then((result) => result.map(checkResultFromStatus)),
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
            <DataTableSkeleton columnCount={3} filterCount={1} withExport />
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
              <DataTableSearchInput
                value={tableSearch.q ?? ""}
                onValueChange={tableSearch.onQueryChange}
                placeholder="Search check results"
              />
              <CheckResultsToolbar table={table} />
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
