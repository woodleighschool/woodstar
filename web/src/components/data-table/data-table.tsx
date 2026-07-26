import { flexRender, type Table as TanstackTable } from "@tanstack/react-table";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import type * as React from "react";

import { DataTablePagination } from "@components/data-table/data-table-pagination";
import { TableSurface } from "@components/data-table/table-surface";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { cn } from "@lib/utils";

interface DataTableProps<TData> extends React.ComponentProps<"div"> {
  table: TanstackTable<TData>;
  actionBar?: React.ReactNode;
  empty?: React.ReactNode;
  heading?: React.ReactNode;
}

export function DataTable<TData>({
  table,
  actionBar,
  empty,
  heading,
  children,
  className,
  ...props
}: DataTableProps<TData>) {
  return (
    <div className={cn("min-w-0", className)} {...props}>
      <TableSurface
        heading={heading}
        toolbar={children}
        viewportClassName="max-h-[calc(100svh-23rem)]"
        footer={<DataTablePagination table={table} className="border-t px-3 py-2" />}
      >
        <Table>
          <TableHeader className="sticky top-0 z-10">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  const direction = header.column.getIsSorted();
                  const label = (
                    <span className="truncate">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </span>
                  );

                  const sortIndicator = (
                    <span className="flex shrink-0 flex-col -space-y-1.5" aria-hidden="true">
                      <ChevronUpIcon
                        className={cn(
                          "size-3",
                          direction === "asc"
                            ? "text-foreground"
                            : direction === "desc"
                              ? "text-muted-foreground/25"
                              : "text-muted-foreground/60",
                        )}
                      />
                      <ChevronDownIcon
                        className={cn(
                          "size-3",
                          direction === "desc"
                            ? "text-foreground"
                            : direction === "asc"
                              ? "text-muted-foreground/25"
                              : "text-muted-foreground/60",
                        )}
                      />
                    </span>
                  );

                  return (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      aria-sort={
                        direction === "asc"
                          ? "ascending"
                          : direction === "desc"
                            ? "descending"
                            : "none"
                      }
                    >
                      {header.isPlaceholder ? null : header.column.getCanSort() ? (
                        <button
                          type="button"
                          className="flex size-full cursor-pointer items-center justify-between gap-2 text-left select-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                          onClick={header.column.getToggleSortingHandler()}
                        >
                          {label}
                          {sortIndicator}
                        </button>
                      ) : (
                        label
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={table.getVisibleLeafColumns().length} className="p-0">
                  {empty}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableSurface>
      {actionBar}
    </div>
  );
}
