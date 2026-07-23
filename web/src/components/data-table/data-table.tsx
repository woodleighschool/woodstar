import { flexRender, type Table as TanstackTable } from "@tanstack/react-table";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import type * as React from "react";

import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

interface DataTableProps<TData> extends React.ComponentProps<"div"> {
  table: TanstackTable<TData>;
  actionBar?: React.ReactNode;
  empty?: React.ReactNode;
}

export function DataTable<TData>({
  table,
  actionBar,
  empty,
  children,
  className,
  ...props
}: DataTableProps<TData>) {
  return (
    <div
      className={cn(
        "flex w-full flex-col overflow-hidden rounded-xl border bg-card shadow-sm",
        className,
      )}
      {...props}
    >
      {children ? <div className="border-b p-2">{children}</div> : null}
      <div className="max-h-[calc(100svh-23rem)] overflow-auto *:data-[slot=table-container]:overflow-visible">
        <Table>
          <TableHeader className="sticky top-0 z-10 bg-muted">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  const direction = header.column.getIsSorted();
                  const content = (
                    <>
                      <span className="truncate">
                        {flexRender(header.column.columnDef.header, header.getContext())}
                      </span>
                      {direction === "asc" ? (
                        <ChevronUpIcon
                          className="shrink-0 opacity-60"
                          size={16}
                          aria-hidden="true"
                        />
                      ) : direction === "desc" ? (
                        <ChevronDownIcon
                          className="shrink-0 opacity-60"
                          size={16}
                          aria-hidden="true"
                        />
                      ) : null}
                    </>
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
                          className="flex size-full cursor-pointer items-center justify-between gap-2 text-left underline decoration-dotted underline-offset-4 select-none hover:decoration-solid focus-visible:decoration-solid focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                          onClick={header.column.getToggleSortingHandler()}
                        >
                          {content}
                        </button>
                      ) : (
                        content
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody className="[&_a:not([data-slot=button])]:underline [&_a:not([data-slot=button])]:decoration-dotted [&_a:not([data-slot=button])]:underline-offset-4 [&_a:not([data-slot=button]):focus-visible]:decoration-solid [&_a:not([data-slot=button]):hover]:decoration-solid [&_button:not([data-slot=button])]:underline [&_button:not([data-slot=button])]:decoration-dotted [&_button:not([data-slot=button])]:underline-offset-4 [&_button:not([data-slot=button]):focus-visible]:decoration-solid [&_button:not([data-slot=button]):hover]:decoration-solid">
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
      </div>
      <DataTablePagination table={table} className="border-t px-3 py-2" />
      {actionBar}
    </div>
  );
}
