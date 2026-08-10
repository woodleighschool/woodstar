import type {
  DataTableExportColumn,
  DataTableExportData,
} from "@components/data-table/data-table-export";
import { DataTableRowExpander } from "@components/data-table/data-table-row-expander";
import type { DataTableColumnDef } from "@components/data-table/types";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { TextLink } from "@components/link";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import type { OsqueryReportSnapshot } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";
import { formatRelative } from "@lib/utils";

type SnapshotStatus = OsqueryReportSnapshot["status"];
export type ReportResultStatus = SnapshotStatus | "error" | "stopped";

export type ReportResultRow = {
  host_id: number;
  host_name: string;
  status: ReportResultStatus;
  rows: Record<string, string>[];
  result_row_count: number;
  returned_row_count: number;
  collected_at?: string;
  updated_at?: string;
  error?: string;
};

export const REPORT_SNAPSHOT_STATUS_VALUES = ["collected", "pending"] as const;

export const REPORT_SNAPSHOT_STATUS_OPTIONS = [
  { label: "Collected", value: "collected" },
  { label: "Pending", value: "pending" },
] satisfies { label: string; value: SnapshotStatus }[];

const REPORT_RESULT_STATUSES = {
  collected: { name: "Collected", variant: "success" },
  pending: { name: "Pending", variant: "default" },
  error: { name: "Error", variant: "error" },
  stopped: { name: "Stopped", variant: "default" },
} satisfies StatusMetadataMap<ReportResultStatus>;

export function reportResultFromSnapshot(snapshot: OsqueryReportSnapshot): ReportResultRow {
  return {
    host_id: snapshot.host_id,
    host_name: snapshot.host_name,
    status: snapshot.status,
    rows: snapshot.rows,
    result_row_count: snapshot.result_row_count,
    returned_row_count: snapshot.returned_row_count,
    collected_at: snapshot.collected_at,
  };
}

export function createReportResultColumns({
  timestamp,
  includeError = false,
}: {
  timestamp: "collected" | "reported";
  includeError?: boolean;
}): DataTableColumnDef<ReportResultRow>[] {
  const timestampColumn: DataTableColumnDef<ReportResultRow> =
    timestamp === "collected"
      ? {
          id: "collected_at",
          accessorKey: "collected_at",
          header: () => "Last Collected",
          cell: ({ row }) =>
            row.original.collected_at ? formatRelative(row.original.collected_at) : "-",
        }
      : {
          id: "updated_at",
          accessorKey: "updated_at",
          header: () => "Last Reported",
          cell: ({ row }) =>
            row.original.updated_at ? formatRelative(row.original.updated_at) : "-",
        };

  const columns: DataTableColumnDef<ReportResultRow>[] = [
    {
      id: "expand",
      header: () => <span className="sr-only">Expand</span>,
      cell: ({ row }) => <DataTableRowExpander row={row} label={row.original.host_name} />,
      enableSorting: false,
      size: 44,
      minSize: 44,
      maxSize: 44,
      enableResizing: false,
    },
    {
      id: "host_name",
      accessorKey: "host_name",
      header: () => "Host",
      cell: ({ row }) => (
        <TextLink to="/hosts/$id" params={{ id: String(row.original.host_id) }}>
          {row.original.host_name}
        </TextLink>
      ),
    },
    {
      id: "status",
      accessorKey: "status",
      header: () => "Status",
      enableColumnFilter: true,
      cell: ({ row }) => <ReportResultStatus row={row.original} />,
    },
    timestampColumn,
    {
      id: "result_row_count",
      accessorKey: "result_row_count",
      header: () => "Result Rows",
      cell: ({ row }) => resultRowCountLabel(row.original),
    },
  ];
  if (includeError) {
    columns.push({
      id: "error",
      accessorKey: "error",
      header: () => "Error",
      cell: ({ row }) => row.original.error || "-",
    });
  }
  return columns;
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

function snapshotStatus(row: ReportResultRow): ReportResultStatus {
  return row.status;
}

export function snapshotStatusLabel(row: ReportResultRow): string {
  return REPORT_RESULT_STATUSES[snapshotStatus(row)].name;
}

export function ReportResultStatus({ row }: { row: ReportResultRow }) {
  return <EnumStatusIndicator value={snapshotStatus(row)} metadata={REPORT_RESULT_STATUSES} />;
}

export function resultRowCountLabel(row: ReportResultRow): string {
  if (row.status !== "collected") return "-";
  if (row.returned_row_count === row.result_row_count) return String(row.result_row_count);
  return `${row.returned_row_count} of ${row.result_row_count}`;
}

export function serializeSnapshots<T extends Pick<ReportResultRow, "rows">>(
  rows: T[],
  metadataColumns: DataTableExportColumn<T>[],
): DataTableExportData {
  const columnNames = resultColumnNames(rows.flatMap((row) => row.rows));
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

function resultValue(value: string | undefined) {
  if (value == null || value === "") return "-";
  return value;
}
