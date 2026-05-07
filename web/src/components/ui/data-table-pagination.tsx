import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

const PER_PAGE_OPTIONS = [25, 50, 100, 200] as const;

interface DataTablePaginationProps {
  page: number;
  perPage: number;
  totalCount: number;
  visibleCount: number;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
}

export function DataTablePagination({
  page,
  perPage,
  totalCount,
  visibleCount,
  onPageChange,
  onPerPageChange,
}: DataTablePaginationProps) {
  const pageCount = Math.max(1, Math.ceil(totalCount / perPage));
  const fromIndex = totalCount === 0 ? 0 : (page - 1) * perPage + 1;
  const toIndex = (page - 1) * perPage + visibleCount;

  return (
    <div className="flex flex-col-reverse items-center justify-between gap-3 px-3 py-2 sm:flex-row">
      <div className="text-muted-foreground text-xs tabular-nums">
        {totalCount === 0 ? "No results" : `${fromIndex}–${toIndex} of ${totalCount}`}
      </div>
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 text-xs">
          <span className="text-muted-foreground">Rows per page</span>
          <Select value={String(perPage)} onValueChange={(v) => onPerPageChange(Number(v))}>
            <SelectTrigger size="sm" className="h-7 w-[64px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PER_PAGE_OPTIONS.map((n) => (
                <SelectItem key={n} value={String(n)}>
                  {n}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="text-muted-foreground text-xs tabular-nums">
          Page {page} of {pageCount}
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={page <= 1}
            onClick={() => onPageChange(1)}
            aria-label="First page"
          >
            <ChevronsLeft className="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={page <= 1}
            onClick={() => onPageChange(page - 1)}
            aria-label="Previous page"
          >
            <ChevronLeft className="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={page >= pageCount}
            onClick={() => onPageChange(page + 1)}
            aria-label="Next page"
          >
            <ChevronRight className="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="size-7"
            disabled={page >= pageCount}
            onClick={() => onPageChange(pageCount)}
            aria-label="Last page"
          >
            <ChevronsRight className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
