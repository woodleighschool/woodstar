import type { Table as TanStackTable } from "@tanstack/react-table";
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PAGE_SIZE_OPTIONS } from "@/lib/pagination";

interface DataTablePaginationProps<TData> {
  table: TanStackTable<TData>;
  totalCount: number;
  perPageOptions?: readonly number[];
}

export function DataTablePagination<TData>({
  table,
  totalCount,
  perPageOptions = PAGE_SIZE_OPTIONS,
}: DataTablePaginationProps<TData>) {
  const { pageIndex, pageSize } = table.getState().pagination;
  const pageCount = Math.max(1, table.getPageCount());
  const visibleCount = table.getRowModel().rows.length;
  const fromIndex = totalCount === 0 ? 0 : pageIndex * pageSize + 1;
  const toIndex = pageIndex * pageSize + visibleCount;

  return (
    <div className="flex flex-col-reverse items-center justify-between gap-3 px-3 py-2 sm:flex-row">
      <div className="text-muted-foreground text-xs tabular-nums">
        {totalCount === 0 ? "No results" : `${fromIndex}–${toIndex} of ${totalCount}`}
      </div>
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 text-xs">
          <span className="text-muted-foreground">Rows per page</span>
          <Select value={String(pageSize)} onValueChange={(value) => table.setPageSize(Number(value))}>
            <SelectTrigger size="sm" className="w-[76px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {perPageOptions.map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="text-muted-foreground text-xs tabular-nums">
          Page {pageIndex + 1} of {pageCount}
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={!table.getCanPreviousPage()}
            onClick={() => table.firstPage()}
          >
            <ChevronsLeft />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={!table.getCanPreviousPage()}
            onClick={() => table.previousPage()}
          >
            <ChevronLeft />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={!table.getCanNextPage()}
            onClick={() => table.nextPage()}
          >
            <ChevronRight />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={!table.getCanNextPage()}
            onClick={() => table.lastPage()}
          >
            <ChevronsRight />
          </Button>
        </div>
      </div>
    </div>
  );
}
