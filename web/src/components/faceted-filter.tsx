import { PlusCircle, X } from "lucide-react";
import * as React from "react";

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

export interface FacetedFilterOption {
  label: string;
  value: string;
  count?: number;
  icon?: React.FC<React.SVGProps<SVGSVGElement>>;
}

export function FacetedFilter({
  title,
  options,
  value = [],
  multiple = options.length > 2,
  onValueChange,
}: {
  title?: string;
  options: FacetedFilterOption[];
  value?: readonly string[];
  multiple?: boolean;
  onValueChange: (value: string[]) => void;
}) {
  const selectedValues = React.useMemo(() => new Set(value), [value]);
  const selected = Array.from(selectedValues, String);
  const items = options.map((option) => option.value);
  const optionsByValue = React.useMemo(
    () => new Map(options.map((option) => [option.value, option])),
    [options],
  );
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
            aria-label={title ? `Clear ${title.toLowerCase()} filter` : "Clear filter"}
            onClick={() => onValueChange([])}
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
        <ComboboxEmpty>No Results Found</ComboboxEmpty>
        <ComboboxList className="max-h-72">
          {(itemValue) => {
            const option = optionsByValue.get(itemValue);
            if (!option) return null;

            return (
              <ComboboxItem key={option.value} value={option.value}>
                {option.icon ? <option.icon /> : null}
                <span>{option.label}</span>
                {option.count !== undefined ? (
                  <span className="ml-auto pr-5 text-xs tabular-nums">{option.count}</span>
                ) : null}
              </ComboboxItem>
            );
          }}
        </ComboboxList>
      </ComboboxContent>
    </>
  );

  if (multiple) {
    return (
      <Combobox multiple items={items} value={selected} onValueChange={onValueChange}>
        {content}
      </Combobox>
    );
  }

  return (
    <Combobox
      items={items}
      value={selected[0] ?? null}
      onValueChange={(next) => onValueChange(next ? [next] : [])}
    >
      {content}
    </Combobox>
  );
}
