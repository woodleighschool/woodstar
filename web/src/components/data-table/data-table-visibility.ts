import type { ColumnDef } from "@tanstack/react-table";

export function defaultHiddenIds<TData>(columns: ColumnDef<TData, unknown>[]): string[] {
  return columns.filter((column) => !!column.id && column.meta?.defaultHidden).map((column) => column.id as string);
}
