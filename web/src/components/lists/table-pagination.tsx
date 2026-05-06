import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export function TablePagination({
  page,
  perPage,
  totalCount,
  visibleCount,
  onPageChange,
  onPerPageChange,
}: {
  page: number;
  perPage: number;
  totalCount: number;
  visibleCount: number;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
}) {
  const first = totalCount === 0 ? 0 : (page - 1) * perPage + 1;
  const last = Math.min((page - 1) * perPage + visibleCount, totalCount);
  const hasPrevious = page > 1;
  const hasNext = page * perPage < totalCount;

  return (
    <div className="flex flex-col gap-3 border-t px-3 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
      <div className="text-muted-foreground">
        Showing <span className="tabular-nums">{first}</span>-<span className="tabular-nums">{last}</span> of{" "}
        <span className="tabular-nums">{totalCount}</span>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-2 text-muted-foreground">
          <span>Rows</span>
          <Select value={String(perPage)} onValueChange={(value) => onPerPageChange(Number(value))}>
            <SelectTrigger className="h-8 w-[5.5rem]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[25, 50, 100, 200].map((value) => (
                <SelectItem key={value} value={String(value)}>
                  {value}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" size="sm" disabled={!hasPrevious} onClick={() => onPageChange(page - 1)}>
          Previous
        </Button>
        <Button variant="outline" size="sm" disabled={!hasNext} onClick={() => onPageChange(page + 1)}>
          Next
        </Button>
      </div>
    </div>
  );
}
