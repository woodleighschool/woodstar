import { getRouteApi, useParams } from "@tanstack/react-router";
import type { ColumnDef, Table } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTableClient } from "@components/data-table/data-table-client";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { LabelTargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import { Skeleton } from "@components/ui/skeleton";
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
import { useCheck, useCheckResults } from "./queries";

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
    filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
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

function renderCheckResultsToolbar(table: Table<OsqueryCheckHostStatus>) {
  return <CheckResultsToolbar table={table} />;
}

export function CheckDetailPage() {
  const { id: checkId } = useParams({
    from: "/_authenticated/osquery/checks/$id",
  });
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
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
  const results = useCheckResults(id, { status });
  const rows = results.data ?? [];

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

  return (
    <PageShell>
      <PageHeader
        title="Check Details"
        description={check.data.description || undefined}
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

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={check.data.name} />
        <KeyValueRow label="Passing" value={formatHostCount(check.data.passing_host_count)} />
        <KeyValueRow label="Failing" value={formatHostCount(check.data.failing_host_count)} />
      </KeyValueSection>

      <LabelTargetDetails targets={check.data.targets} />

      {results.error ? (
        <QueryError
          title="Failed to load check results"
          error={results.error}
          onRetry={() => void results.refetch()}
        />
      ) : results.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTableClient
          title="Results"
          columns={resultColumns}
          data={rows}
          exportOptions={{
            filename: `osquery-check-${id}-results`,
            columns: resultExportColumns,
          }}
          searchPlaceholder="Search check results"
          tableState={tableSearch}
          toolbar={renderCheckResultsToolbar}
          empty={
            <PanelEmptyState>
              {status ? "No check results match this status" : "No check results yet"}
            </PanelEmptyState>
          }
        />
      )}

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
