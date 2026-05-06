import { ArrowDown, ArrowUp, ChevronsUpDown } from "lucide-react";
import type React from "react";

import { nextSortState, type SortState } from "@/components/lists/sort-state";
import { Button } from "@/components/ui/button";
import { TableHead } from "@/components/ui/table";
import { cn } from "@/lib/utils";

export function SortableTableHead({
  orderKey,
  active,
  onSort,
  align = "left",
  children,
  className,
}: {
  orderKey: string;
  active: SortState;
  onSort: (next: SortState) => void;
  align?: "left" | "right";
  children: React.ReactNode;
  className?: string;
}) {
  const isActive = active.orderKey === orderKey;
  const Icon = !isActive ? ChevronsUpDown : active.orderDirection === "desc" ? ArrowDown : ArrowUp;

  return (
    <TableHead className={className}>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className={cn("h-auto px-0 py-0 font-medium", align === "right" && "ml-auto")}
        onClick={() => onSort(nextSortState(active, orderKey))}
      >
        {children}
        <Icon data-icon="inline-end" />
      </Button>
    </TableHead>
  );
}
