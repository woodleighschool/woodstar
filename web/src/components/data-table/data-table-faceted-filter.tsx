import * as React from "react";

import type { DataTableColumn, DataTableRowData, Option } from "@components/data-table/types";
import { FacetedFilter } from "@components/faceted-filter";

interface DataTableFacetedFilterProps<TData extends DataTableRowData, TValue> {
  column?: DataTableColumn<TData, TValue>;
  title?: string;
  options: Option[];
}

export function DataTableFacetedFilter<TData extends DataTableRowData, TValue>({
  column,
  title,
  options,
}: DataTableFacetedFilterProps<TData, TValue>) {
  const columnFilterValue = column?.getFilterValue();
  const value = React.useMemo(
    () => (Array.isArray(columnFilterValue) ? columnFilterValue.map(String) : []),
    [columnFilterValue],
  );

  function setFilter(next: string[]) {
    column?.setFilterValue(next.length > 0 ? next : undefined);
  }
  return (
    <FacetedFilter
      title={title}
      options={options}
      value={value}
      multiple={options.length > 2}
      onValueChange={setFilter}
    />
  );
}
