import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTableClient } from "@components/data-table/data-table-client";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableRowExpander } from "@components/data-table/data-table-row-expander";
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
import { LiveRunButton, ShowQueryButton } from "@features/osquery/live/query-actions";
import { parseRouteID } from "@lib/route-params";
import { formatInterval, formatRelative } from "@lib/utils";

import { ReportDeleteDialog } from "./delete-dialog";
import { useReport, useReportSnapshots } from "./queries";
import {
  reportSnapshotRows,
  type ReportSnapshotTableRow,
  resultColumnNames,
  serializeSnapshots,
  SnapshotResultRows,
  snapshotSearchText,
  snapshotStatusLabel,
  SnapshotStatusBadge,
} from "./query-results";

const reportSnapshotColumns: ColumnDef<ReportSnapshotTableRow>[] = [
  {
    id: "expand",
    header: () => <span className="sr-only">Expand</span>,
    cell: ({ row }) => <DataTableRowExpander row={row} label={row.original.hostName} />,
    enableSorting: false,
    size: 44,
  },
  {
    id: "hostName",
    accessorKey: "hostName",
    header: () => "Host",
    cell: ({ row }) => (
      <Link to="/hosts/$id" params={{ id: String(row.original.hostId) }}>
        {row.original.hostName}
      </Link>
    ),
  },
  {
    id: "status",
    accessorFn: snapshotStatusLabel,
    header: () => "Collection",
    cell: ({ row }) => <SnapshotStatusBadge row={row.original} />,
  },
  {
    id: "collectedAt",
    accessorKey: "collectedAt",
    header: () => "Collected",
    cell: ({ row }) => (row.original.collectedAt ? formatRelative(row.original.collectedAt) : "-"),
  },
  {
    id: "rowCount",
    accessorFn: (row) => row.rows.length,
    header: () => "Result Rows",
  },
];

export function ReportDetailPage() {
  const { id: reportId } = useParams({
    from: "/_authenticated/osquery/reports/$id",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleteOpen, setDeleteOpen] = useState(false);
  const id = parseRouteID(reportId);
  const report = useReport(id);
  const snapshots = useReportSnapshots(id);

  if (id === null) {
    return (
      <QueryGate title="Failed to load report" error={{ message: "Report route is invalid." }} />
    );
  }

  if (report.error) {
    return (
      <QueryGate
        title="Failed to load report"
        error={report.error}
        onRetry={() => void report.refetch()}
      />
    );
  }

  if (!report.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  const rows = reportSnapshotRows(snapshots.data);
  const columnNames = resultColumnNames(rows.flatMap((row) => row.rows));
  const exportMetadata: DataTableExportOptions<ReportSnapshotTableRow>["columns"] = [
    { header: "Host", value: (row) => row.hostName },
    { header: "Collection Status", value: snapshotStatusLabel },
    { header: "Collected At", value: (row) => row.collectedAt },
  ];
  const exportOptions: DataTableExportOptions<ReportSnapshotTableRow> = {
    filename: `osquery-report-${id}-results`,
    columns: exportMetadata,
    serializeRows: (exportRows) => serializeSnapshots(exportRows, exportMetadata, columnNames),
  };

  return (
    <PageShell>
      <PageHeader
        title="Report Details"
        description={report.data.description || undefined}
        meta={`Edited ${formatRelative(report.data.updated_at)}`}
        actions={
          <>
            {isAdmin ? (
              <>
                <Button
                  size="sm"
                  render={<Link to="/osquery/reports/$id/edit" params={{ id: reportId }} />}
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
            <ShowQueryButton sql={report.data.query} />
            <LiveRunButton to="/osquery/reports/$id/live" params={{ id: reportId }} />
          </>
        }
      />

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={report.data.name} />
        <KeyValueRow
          label="Interval"
          value={
            report.data.schedule_interval
              ? `Every ${formatInterval(report.data.schedule_interval)}`
              : "Off"
          }
        />
        <KeyValueRow label="Minimum Osquery" value={report.data.min_osquery_version || "Any"} />
      </KeyValueSection>

      <LabelTargetDetails targets={report.data.targets} />

      {snapshots.error ? (
        <QueryError
          title="Failed to load report results"
          error={snapshots.error}
          onRetry={() => void snapshots.refetch()}
        />
      ) : snapshots.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTableClient
          title="Results"
          columns={reportSnapshotColumns}
          data={rows}
          exportOptions={exportOptions}
          getRowCanExpand={(row) => row.original.rows.length > 0}
          getRowId={(row) => row.id}
          getSearchText={snapshotSearchText}
          initialSorting={[{ id: "hostName", desc: false }]}
          renderSubRow={(row) => (
            <SnapshotResultRows rows={row.original.rows} columnNames={columnNames} />
          )}
          searchPlaceholder="Search hosts and results"
          empty={<PanelEmptyState>No targeted hosts</PanelEmptyState>}
        />
      )}

      {isAdmin ? (
        <ReportDeleteDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          report={report.data}
          onDeleted={() => void navigate({ to: "/osquery/reports" })}
        />
      ) : null}
    </PageShell>
  );
}
