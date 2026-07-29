import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTableClient } from "@components/data-table/data-table-client";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
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
import { useReport, useReportResults } from "./queries";
import {
  reportRows,
  reportTableColumns,
  type ReportTableRow,
  resultColumnNames,
  resultValue,
} from "./query-results";

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
  const results = useReportResults(id);

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

  const rows = reportRows(results.data);
  const columnNames = resultColumnNames(rows);
  const resultColumns: ColumnDef<ReportTableRow>[] = columnNames.map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: () => name,
    cell: ({ row }) => resultValue(row.original.columns[name]),
  }));
  const columns = [...reportTableColumns(), ...resultColumns];
  const exportOptions: DataTableExportOptions<ReportTableRow> = {
    filename: `osquery-report-${id}-results`,
    columns: [
      { header: "Host", value: (row) => row.hostName },
      { header: "Last Fetched", value: (row) => row.lastFetched },
      ...columnNames.map((name) => ({
        header: name,
        value: (row: ReportTableRow) => row.columns[name],
      })),
    ],
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

      {results.error ? (
        <QueryError
          title="Failed to load report results"
          error={results.error}
          onRetry={() => void results.refetch()}
        />
      ) : results.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTableClient
          title="Results"
          columns={columns}
          data={rows}
          exportOptions={exportOptions}
          initialSorting={[{ id: "hostName", desc: false }]}
          searchPlaceholder="Search report results"
          empty={<PanelEmptyState>No report results yet</PanelEmptyState>}
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
