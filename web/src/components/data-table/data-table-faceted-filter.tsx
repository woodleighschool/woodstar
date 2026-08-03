import type { Column } from "@tanstack/react-table";
import { PlusCircle, X } from "lucide-react";
import * as React from "react";

import type { Option } from "@components/data-table/types";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import { ButtonGroup } from "@components/ui/button-group";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
} from "@components/ui/combobox";
import { Separator } from "@components/ui/separator";

interface DataTableFacetedFilterProps<TData, TValue> {
  column?: Column<TData, TValue>;
  title?: string;
  options: Option[];
}

export function DataTableFacetedFilter<TData, TValue>({
  column,
  title,
  options,
}: DataTableFacetedFilterProps<TData, TValue>) {
  const columnFilterValue = column?.getFilterValue();
  const selectedValues = React.useMemo(
    () => new Set(Array.isArray(columnFilterValue) ? columnFilterValue : []),
    [columnFilterValue],
  );

  function setMultipleFilter(next: string[]) {
    column?.setFilterValue(next.length > 0 ? next : undefined);
  }

  function setSingleFilter(next: string | null) {
    column?.setFilterValue(next ? [next] : undefined);
  }

  function resetFilter() {
    column?.setFilterValue(undefined);
  }

  const selected = Array.from(selectedValues, String);
  const items = options.map((option) => option.value);
  const multiple = options.length > 2;
  const content = (
    <>
      <ButtonGroup>
        <ComboboxTrigger
          render={<Button variant="outline" size="sm" className="h-8 border-dashed font-normal" />}
        >
          <PlusCircle data-icon="inline-start" />
          {title}
          {selectedValues.size > 0 ? (
            <>
              <Separator orientation="vertical" className="mx-0.5 h-4" />
              {selectedValues.size > 2 ? (
                <Badge variant="secondary" className="rounded-sm px-1 font-normal">
                  {selectedValues.size} selected
                </Badge>
              ) : (
                options
                  .filter((option) => selectedValues.has(option.value))
                  .map((option) => (
                    <Badge
                      key={option.value}
                      variant="secondary"
                      className="rounded-sm px-1 font-normal"
                    >
                      {option.label}
                    </Badge>
                  ))
              )}
            </>
          ) : null}
        </ComboboxTrigger>
        {selectedValues.size > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            className="size-8"
            onClick={resetFilter}
          >
            <X />
          </Button>
        ) : null}
      </ButtonGroup>
      <ComboboxContent className="w-64 min-w-64">
        <ComboboxInput
          showTrigger={false}
          placeholder={title ? `Search ${title.toLowerCase()}...` : "Search..."}
        />
        <ComboboxEmpty>No results found.</ComboboxEmpty>
        <ComboboxList className="max-h-72">
          {options.map((option) => (
            <ComboboxItem key={option.value} value={option.value}>
              {option.icon ? <option.icon /> : null}
              <span>{option.label}</span>
              {option.count !== undefined ? (
                <span className="ml-auto pr-5 text-xs tabular-nums">{option.count}</span>
              ) : null}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </>
  );

  if (multiple) {
    return (
      <Combobox multiple items={items} value={selected} onValueChange={setMultipleFilter}>
        {content}
      </Combobox>
    );
  }

  return (
    <Combobox items={items} value={selected[0] ?? null} onValueChange={setSingleFilter}>
      {content}
    </Combobox>
  );
}
