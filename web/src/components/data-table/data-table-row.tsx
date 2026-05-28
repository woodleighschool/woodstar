import { useNavigate } from "@tanstack/react-router";
import { flexRender, type Row as TanStackRow } from "@tanstack/react-table";
import type { ComponentProps } from "react";

import { TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";

const INTERACTIVE_SELECTOR = [
  "a",
  "button",
  "input",
  "label",
  "select",
  "textarea",
  "[data-slot=dropdown-menu-content]",
].join(", ");

export interface DataTableBodyRowProps<TData> {
  row: TanStackRow<TData>;
  enableRowSelection?: boolean;
  rowHref?: (row: TData) => { to: string; params?: Record<string, string> };
  onRowClick?: (row: TData) => void;
}

export function DataTableBodyRow<TData>({
  row,
  enableRowSelection = false,
  rowHref,
  onRowClick,
  ref,
  style,
}: DataTableBodyRowProps<TData> & Pick<ComponentProps<"tr">, "ref" | "style">) {
  const navigate = useNavigate();
  const linkProps = rowHref?.(row.original);
  const hasRowAction = linkProps !== undefined || onRowClick !== undefined;
  const firstDataIndex = enableRowSelection ? 1 : 0;

  return (
    <TableRow
      ref={ref}
      style={style}
      className={cn(hasRowAction && "cursor-pointer")}
      onClick={
        hasRowAction
          ? (e) => {
              const target = e.target as HTMLElement;
              if (target.closest(INTERACTIVE_SELECTOR)) return;
              if (window.getSelection()?.toString()) return;
              if (linkProps !== undefined) {
                void navigate({ to: linkProps.to, params: linkProps.params });
              } else {
                onRowClick?.(row.original);
              }
            }
          : undefined
      }
    >
      {row.getVisibleCells().map((cell, i) => (
        <TableCell
          key={cell.id}
          className={cn(cell.column.columnDef.meta?.cellClassName, i === firstDataIndex && "font-medium")}
        >
          {flexRender(cell.column.columnDef.cell, cell.getContext())}
        </TableCell>
      ))}
    </TableRow>
  );
}
