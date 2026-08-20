import { format, isValid, parseISO, subMonths } from "date-fns";
import { X } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DateRangePicker } from "@components/date-range-picker";
import { FacetedFilter } from "@components/faceted-filter";
import { Button } from "@components/ui/button";
import type { ActivityEvent } from "@lib/api";
import { isOneOf } from "@lib/utils";

import {
  ACTIVITY_ACTION_OPTIONS,
  ACTIVITY_ACTION_VALUES,
  ACTIVITY_AREA_OPTIONS,
  ACTIVITY_AREA_VALUES,
  ACTIVITY_SCOPE_VALUES,
  type ActivityScope,
} from "./metadata";

const ACTIVITY_SCOPE_OPTIONS = [
  { value: "user", label: "Administrators" },
  { value: "system", label: "System" },
] as const;

export interface ActivityFilterState {
  q?: string;
  scope: ActivityScope;
  area?: ActivityEvent["area"];
  action?: ActivityEvent["action"];
  from?: string;
  to?: string;
}

export function ActivityFilters({
  value,
  loading,
  showReset,
  onChange,
  onReset,
}: {
  value: ActivityFilterState;
  loading: boolean;
  showReset: boolean;
  onChange: (next: Partial<ActivityFilterState>) => void;
  onReset: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2" aria-busy={loading || undefined}>
      <DataTableSearchInput
        value={value.q ?? ""}
        onValueChange={(q) => onChange({ q })}
        loading={loading}
        placeholder="Search actors or resources"
        aria-label="Search activity"
      />
      <FacetedFilter
        title="Actor"
        options={[...ACTIVITY_SCOPE_OPTIONS]}
        value={value.scope === "all" ? [] : [value.scope]}
        multiple={false}
        onValueChange={(selected) => {
          const scope = selected.at(-1) ?? "all";
          if (isOneOf(scope, ACTIVITY_SCOPE_VALUES)) onChange({ scope });
        }}
      />
      <FacetedFilter
        title="Area"
        options={[...ACTIVITY_AREA_OPTIONS]}
        value={value.area ? [value.area] : []}
        multiple={false}
        onValueChange={(selected) => {
          const area = selected.at(-1);
          if (area === undefined || isOneOf(area, ACTIVITY_AREA_VALUES)) onChange({ area });
        }}
      />
      <FacetedFilter
        title="Action"
        options={[...ACTIVITY_ACTION_OPTIONS]}
        value={value.action ? [value.action] : []}
        multiple={false}
        onValueChange={(selected) => {
          const action = selected.at(-1);
          if (action === undefined || isOneOf(action, ACTIVITY_ACTION_VALUES)) {
            onChange({ action });
          }
        }}
      />
      <DateRangePicker
        value={parseDateRange(value.from, value.to)}
        defaultMonth={subMonths(new Date(), 1)}
        disabled={{ after: new Date() }}
        onValueChange={(range) =>
          onChange({
            from: range?.from ? format(range.from, "yyyy-MM-dd") : undefined,
            to: range?.to ? format(range.to, "yyyy-MM-dd") : undefined,
          })
        }
      />
      {showReset ? (
        <Button type="button" variant="ghost" size="sm" onClick={onReset}>
          <X data-icon="inline-start" />
          Reset
        </Button>
      ) : null}
    </div>
  );
}

function parseDateRange(fromValue: string | undefined, toValue: string | undefined): DateRange {
  const from = parseDate(fromValue);
  const to = parseDate(toValue);
  return { from, to };
}

function parseDate(value: string | undefined): Date | undefined {
  if (!value) return undefined;
  const date = parseISO(value);
  return isValid(date) ? date : undefined;
}
