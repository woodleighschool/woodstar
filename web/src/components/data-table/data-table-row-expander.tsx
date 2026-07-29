import type { Row } from "@tanstack/react-table";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";

import { Button } from "@components/ui/button";

export function DataTableRowExpander<TData>({ row, label }: { row: Row<TData>; label: string }) {
  if (!row.getCanExpand()) return null;

  const expanded = row.getIsExpanded();

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      aria-expanded={expanded}
      aria-label={`${expanded ? "Collapse" : "Expand"} ${label}`}
      onClick={row.getToggleExpandedHandler()}
    >
      {expanded ? <ChevronUpIcon /> : <ChevronDownIcon />}
    </Button>
  );
}
