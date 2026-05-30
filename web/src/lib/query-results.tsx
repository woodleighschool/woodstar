import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";

import { DataTableColumnHeader } from "@/components/data-table";
import type { ReportResult } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export type ReportResultRow = ReportResult;

export type ReportTableRow = {
  id: string;
  reportId: number;
  hostId: number;
  hostName: string;
  lastFetched?: string;
  columns: Record<string, string>;
};

export function reportRows(rows: ReportResultRow[] | null | undefined): ReportTableRow[] {
  return (rows ?? []).map((row, index) => ({
    id: `${row.report_id}-${row.host_id}-${index}`,
    reportId: row.report_id,
    hostId: row.host_id,
    hostName: row.host_name,
    lastFetched: row.last_fetched,
    columns: row.columns,
  }));
}

export function resultColumnNames(rows: ReportTableRow[]): string[] {
  const seen = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row.columns)) {
      seen.add(key);
    }
  }
  return Array.from(seen).sort((a, b) => a.localeCompare(b));
}

export function reportTableColumns(options: { linkHosts?: boolean } = {}): ColumnDef<ReportTableRow>[] {
  return [
    {
      id: "hostName",
      accessorKey: "hostName",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) =>
        options.linkHosts ? (
          <Link to="/hosts/$hostId" params={{ hostId: String(row.original.hostId) }} className="hover:underline">
            {row.original.hostName}
          </Link>
        ) : (
          row.original.hostName
        ),
    },
    {
      id: "lastFetched",
      accessorKey: "lastFetched",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last Fetched" />,
      cell: ({ row }) => (row.original.lastFetched ? formatRelative(row.original.lastFetched) : "-"),
    },
  ];
}

export function resultValue(value: string | undefined) {
  if (value == null || value === "") return "-";
  return value;
}
