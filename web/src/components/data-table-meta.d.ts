import type { RowData } from "@tanstack/table-core";

declare module "@tanstack/table-core" {
  interface ColumnMeta<_TData extends RowData, _TValue> {
    label?: string;
    headClassName?: string;
    cellClassName?: string;
  }
}
