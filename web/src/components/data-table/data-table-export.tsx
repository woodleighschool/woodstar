import { DownloadIcon } from "lucide-react";
import * as React from "react";

import type { DataTableInstance, DataTableRowData } from "@components/data-table/types";
import { Button } from "@components/ui/button";
import { Spinner } from "@components/ui/spinner";
import { toast } from "@components/ui/toast";

type CSVValue = string | number | boolean | null | undefined;

export interface DataTableExportData {
  fields: string[];
  data: CSVValue[][];
}

export interface DataTableExportColumn<TData extends DataTableRowData> {
  header: string;
  value: (row: TData) => CSVValue;
}

export interface DataTableExportOptions<TData extends DataTableRowData> {
  filename: string;
  columns: DataTableExportColumn<TData>[];
  loadRows?: () => Promise<TData[]>;
  serializeRows?: (rows: TData[]) => DataTableExportData;
}

export function DataTableExport<TData extends DataTableRowData>({
  table,
  options,
}: {
  table: DataTableInstance<TData>;
  options: DataTableExportOptions<TData>;
}) {
  const [isExporting, setIsExporting] = React.useState(false);
  const hasRows = table.getRowCount() > 0;

  async function exportCSV() {
    setIsExporting(true);

    try {
      const selectedRows = table.getSelectedRowModel().rows;
      const rows =
        selectedRows.length > 0
          ? selectedRows.map((row) => row.original)
          : options.loadRows
            ? await options.loadRows()
            : table.getFilteredRowModel().rows.map((row) => row.original);

      const { default: Papa } = await import("papaparse");
      const exportData = options.serializeRows?.(rows) ?? {
        fields: options.columns.map((column) => column.header),
        data: rows.map((row) =>
          options.columns.map((column) => {
            const value = column.value(row);
            return value ?? "";
          }),
        ),
      };
      const csv = Papa.unparse(exportData);
      const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");

      link.href = url;
      link.download = `${options.filename}-${new Date().toISOString().slice(0, 10)}.csv`;
      link.hidden = true;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch {
      toast.add({ title: "Failed to Export CSV", type: "error" });
    } finally {
      setIsExporting(false);
    }
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className="h-8 font-normal"
      disabled={!hasRows || isExporting}
      onClick={() => void exportCSV()}
    >
      {isExporting ? (
        <Spinner data-icon="inline-start" />
      ) : (
        <DownloadIcon data-icon="inline-start" />
      )}
      Export
    </Button>
  );
}
