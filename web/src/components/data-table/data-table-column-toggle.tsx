import type { ColumnDef } from "@tanstack/react-table";
import { Columns3 } from "lucide-react";
import type { ComponentProps } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface DataTableColumnToggleProps<TData> {
  columns: ColumnDef<TData, unknown>[];
  hidden: string[];
  onToggle: (id: string) => void;
  variant?: ComponentProps<typeof Button>["variant"];
}

/**
 * Renders a "Columns" dropdown driven by the same ColumnDef array the table uses.
 * Columns marked `meta.alwaysVisible` are skipped; everything else becomes a checkbox.
 */
export function DataTableColumnToggle<TData>({
  columns,
  hidden,
  onToggle,
  variant = "outline",
}: DataTableColumnToggleProps<TData>) {
  const hiddenSet = new Set(hidden);
  const items = columns
    .filter((c) => !!c.id && !c.meta?.alwaysVisible)
    .map((c) => ({
      id: c.id as string,
      label: c.meta?.label ?? (c.id as string),
    }));

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant={variant} size="sm" className="h-8 gap-1.5">
          <Columns3 data-icon="inline-start" />
          Columns
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          {items.map((item) => (
            <DropdownMenuCheckboxItem
              key={item.id}
              checked={!hiddenSet.has(item.id)}
              onCheckedChange={() => onToggle(item.id)}
              onSelect={(e) => e.preventDefault()}
            >
              {item.label}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
