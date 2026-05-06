import { ListFilter } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

export interface FilterOption {
  value: string;
  label: string;
}

export interface FilterGroup {
  id: string;
  label: string;
  options: FilterOption[];
  selected: string[];
  onChange: (selected: string[]) => void;
}

export function FilterPopover({ groups }: { groups: FilterGroup[] }) {
  const filterCount = groups.reduce((sum, group) => sum + group.selected.length, 0);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2">
          <ListFilter className="size-4" />
          Add filters{filterCount > 0 ? ` (${filterCount})` : ""}
        </Button>
      </PopoverTrigger>
      <PopoverContent>
        <div className="flex flex-col gap-4">
          {groups.map((group) => (
            <FilterGroupControls key={group.id} group={group} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function FilterGroupControls({ group }: { group: FilterGroup }) {
  const selected = new Set(group.selected);
  const clearDisabled = group.selected.length === 0;

  const toggle = (value: string) => {
    const next = new Set(selected);
    if (next.has(value)) next.delete(value);
    else next.add(value);
    group.onChange([...next]);
  };

  return (
    <fieldset className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-3">
        <legend className="text-sm font-medium">{group.label}</legend>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          disabled={clearDisabled}
          onClick={() => group.onChange([])}
          type="button"
        >
          Clear
        </Button>
      </div>
      <div className="grid gap-2">
        {group.options.map((option) => {
          const checked = selected.has(option.value);
          return (
            <label key={option.value} className="flex cursor-pointer items-center gap-2 text-sm">
              <Checkbox checked={checked} onCheckedChange={() => toggle(option.value)} />
              <span>{option.label}</span>
            </label>
          );
        })}
      </div>
    </fieldset>
  );
}
