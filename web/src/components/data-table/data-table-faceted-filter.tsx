import type { Column } from "@tanstack/react-table";
import { PlusCircle, X } from "lucide-react";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
  useComboboxAnchor,
} from "@/components/ui/combobox";
import { Separator } from "@/components/ui/separator";
import type { Option } from "@/types/data-table";

interface DataTableFacetedFilterProps<TData, TValue> {
  column?: Column<TData, TValue>;
  title?: string;
  options: Option[];
  multiple?: boolean;
}

export function DataTableFacetedFilter<TData, TValue>({
  column,
  title,
  options,
  multiple = false,
}: DataTableFacetedFilterProps<TData, TValue>) {
  const [open, setOpen] = React.useState(false);
  const anchorRef = useComboboxAnchor();
  const columnFilterValue = column?.getFilterValue();
  const selectedValues = React.useMemo(
    () => new Set(Array.isArray(columnFilterValue) ? columnFilterValue : []),
    [columnFilterValue],
  );

  function setFilter(next: string[]) {
    if (!column) return;

    if (multiple) {
      column.setFilterValue(next.length > 0 ? next : undefined);
      return;
    }

    const added = next.find((value) => !selectedValues.has(value));
    const value = added ?? next[0];
    column.setFilterValue(value ? [value] : undefined);
    setOpen(false);
  }

  function resetFilter() {
    column?.setFilterValue(undefined);
  }

  const selected = Array.from(selectedValues, String);

  return (
    <Combobox
      multiple
      items={options.map((option) => option.value)}
      value={selected}
      onValueChange={setFilter}
      open={open}
      onOpenChange={setOpen}
    >
      <ButtonGroup ref={anchorRef}>
        <ComboboxTrigger
          render={<Button variant="outline" size="sm" className="border-dashed font-normal" />}
        >
          <PlusCircle />
          {title}
          {selectedValues.size > 0 ? (
            <>
              <Separator orientation="vertical" className="mx-0.5 h-4" />
              <Badge variant="secondary" className="rounded-sm px-1 font-normal lg:hidden">
                {selectedValues.size}
              </Badge>
              <span className="hidden items-center gap-1 lg:flex">
                {selectedValues.size > 2 ? (
                  <Badge variant="secondary" className="rounded-sm px-1 font-normal">
                    {selectedValues.size} selected
                  </Badge>
                ) : (
                  options
                    .filter((option) => selectedValues.has(option.value))
                    .map((option) => (
                      <Badge
                        variant="secondary"
                        key={option.value}
                        className="rounded-sm px-1 font-normal"
                      >
                        {option.label}
                      </Badge>
                    ))
                )}
              </span>
            </>
          ) : null}
        </ComboboxTrigger>
        {selectedValues.size > 0 ? (
          <Button type="button" variant="outline" size="icon-sm" onClick={resetFilter}>
            <X />
          </Button>
        ) : null}
      </ButtonGroup>
      <ComboboxContent anchor={anchorRef} className="w-56">
        <ComboboxInput placeholder={title ? `Search ${title.toLowerCase()}...` : "Search..."} />
        <ComboboxEmpty>No results found.</ComboboxEmpty>
        <ComboboxList className="max-h-72">
          {options.map((option) => (
            <ComboboxItem key={option.value} value={option.value}>
              {option.icon ? <option.icon /> : null}
              <span className="truncate">{option.label}</span>
              {option.count !== undefined ? (
                <span className="ml-auto pr-5 font-mono text-xs">{option.count}</span>
              ) : null}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
