import type {
  DataTableExportColumn,
  DataTableExportData,
} from "@components/data-table/data-table-export";
import { Badge } from "@components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import type { OsqueryReportSnapshot } from "@lib/api";

export type ReportSnapshotTableRow = {
  id: string;
  reportId: number;
  reportName: string;
  reportDescription?: string;
  hostId: number;
  hostName: string;
  status: OsqueryReportSnapshot["status"];
  collectedAt?: string;
  rows: Record<string, string>[];
};

type SnapshotStatus = OsqueryReportSnapshot["status"];

export const REPORT_SNAPSHOT_STATUS_VALUES = ["collected", "pending"] as const;

export const REPORT_SNAPSHOT_STATUS_OPTIONS = [
  { label: "Collected", value: "collected" },
  { label: "Pending", value: "pending" },
] satisfies { label: string; value: SnapshotStatus }[];

export function parseReportSnapshotStatus(value: unknown): SnapshotStatus | undefined {
  if (typeof value !== "string") return undefined;
  return REPORT_SNAPSHOT_STATUS_OPTIONS.find((option) => option.value === value)?.value;
}

export function reportSnapshotRows(
  snapshots: OsqueryReportSnapshot[] | null | undefined,
): ReportSnapshotTableRow[] {
  return (snapshots ?? []).map((snapshot) => ({
    id: `${snapshot.report_id}-${snapshot.host_id}`,
    reportId: snapshot.report_id,
    reportName: snapshot.report_name,
    reportDescription: snapshot.report_description,
    hostId: snapshot.host_id,
    hostName: snapshot.host_name,
    status: snapshot.status,
    collectedAt: snapshot.collected_at,
    rows: snapshot.rows,
  }));
}

export function resultColumnNames(rows: Record<string, string>[]): string[] {
  const seen = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row)) {
      seen.add(key);
    }
  }
  return Array.from(seen).toSorted((a, b) => a.localeCompare(b));
}

export function snapshotStatus(row: ReportSnapshotTableRow): SnapshotStatus {
  return row.status;
}

export function snapshotStatusLabel(row: ReportSnapshotTableRow): string {
  const labels: Record<SnapshotStatus, string> = {
    pending: "Pending",
    collected: "Collected",
  };
  return labels[snapshotStatus(row)];
}

export function SnapshotStatusBadge({ row }: { row: ReportSnapshotTableRow }) {
  const status = snapshotStatus(row);
  const variant = status === "collected" ? "success" : "outline";

  return <Badge variant={variant}>{snapshotStatusLabel(row)}</Badge>;
}

export function snapshotSearchText(row: ReportSnapshotTableRow): string {
  return [
    row.reportName,
    row.reportDescription,
    row.hostName,
    snapshotStatusLabel(row),
    row.collectedAt,
    ...row.rows.flatMap((result) => Object.entries(result).flat()),
  ]
    .filter(Boolean)
    .join("\n");
}

export function serializeSnapshots(
  rows: ReportSnapshotTableRow[],
  metadataColumns: DataTableExportColumn<ReportSnapshotTableRow>[],
  columnNames: string[],
): DataTableExportData {
  return {
    fields: [...metadataColumns.map((column) => column.header), ...columnNames],
    data: rows.flatMap((snapshot) => {
      const metadata = metadataColumns.map((column) => column.value(snapshot));
      if (snapshot.rows.length === 0) {
        return [[...metadata, ...columnNames.map(() => "")]];
      }
      return snapshot.rows.map((result) => [
        ...metadata,
        ...columnNames.map((name) => result[name] ?? ""),
      ]);
    }),
  };
}

export function SnapshotResultRows({
  rows,
  columnNames = resultColumnNames(rows),
}: {
  rows: Record<string, string>[];
  columnNames?: string[];
}) {
  if (columnNames.length === 0) {
    return (
      <div className="px-12 py-3 text-sm text-muted-foreground">
        {rows.length} result {rows.length === 1 ? "row has" : "rows have"} no columns.
      </div>
    );
  }

  return (
    <div className="py-2 pr-3 pl-10">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columnNames.map((name) => (
              <TableHead key={name}>{name}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow key={index}>
              {columnNames.map((name) => (
                <TableCell key={name}>{resultValue(row[name])}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export function resultValue(value: string | undefined) {
  if (value == null || value === "") return "-";
  return value;
}
