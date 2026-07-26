import { useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil } from "lucide-react";

import { DataTableClient } from "@components/data-table/data-table-client";
import { KeyValueGrid, KeyValueItem } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent } from "@components/ui/card";
import { Skeleton } from "@components/ui/skeleton";
import { useAuth } from "@features/auth/queries";
import { LiveRunButton, ShowQueryButton } from "@features/osquery/live/query-actions";
import { parseRouteID } from "@lib/route-params";
import { formatInterval, formatRelative } from "@lib/utils";

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
  const { user } = useAuth();
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
  const resultColumns: ColumnDef<ReportTableRow>[] = resultColumnNames(rows).map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: () => name,
    cell: ({ row }) => resultValue(row.original.columns[name]),
  }));
  const columns = [...reportTableColumns(), ...resultColumns];

  return (
    <PageShell>
      <PageHeader
        title={report.data.name}
        description={report.data.description || undefined}
        actions={
          <>
            {user?.role === "admin" ? (
              <Button
                size="sm"
                render={<Link to="/osquery/reports/$id/edit" params={{ id: reportId }} />}
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
            ) : null}
            <ShowQueryButton sql={report.data.query} />
            <LiveRunButton to="/osquery/reports/$id/live" params={{ id: reportId }} />
          </>
        }
      />

      <Card>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem
              label="Interval"
              value={
                report.data.schedule_interval
                  ? `Every ${formatInterval(report.data.schedule_interval)}`
                  : "Off"
              }
            />
            <KeyValueItem
              label="Minimum Osquery"
              value={report.data.min_osquery_version || "Any"}
            />
            <KeyValueItem label="Updated" value={formatRelative(report.data.updated_at)} />
          </KeyValueGrid>
        </CardContent>
      </Card>

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
          columns={columns}
          data={rows}
          initialSorting={[{ id: "hostName", desc: false }]}
          searchPlaceholder="Search report results"
          empty={<PanelEmptyState>No report results yet</PanelEmptyState>}
        />
      )}
    </PageShell>
  );
}
