import type { ReactNode } from "react";
import { useMemo, useState } from "react";

import {
  Combobox,
  ComboboxAnchor,
  ComboboxCancel,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxSeparator,
  ComboboxTrigger,
} from "@/components/ui/combobox";

export function FreeTextCombobox<TItem>({
  id,
  name,
  value,
  items,
  placeholder,
  invalid,
  disabled,
  itemToStringValue,
  freeTextItem,
  itemKey,
  itemDisabled,
  renderItem,
  onBlur,
  onChange,
  onSelectItem,
}: {
  id?: string;
  name?: string;
  value: string;
  items: TItem[];
  placeholder?: string;
  invalid?: boolean;
  disabled?: boolean;
  itemToStringValue: (item: TItem) => string;
  freeTextItem: (value: string) => TItem;
  itemKey?: (item: TItem) => string;
  itemDisabled?: (item: TItem) => boolean;
  renderItem?: (item: TItem) => ReactNode;
  onBlur?: () => void;
  onChange: (value: string) => void;
  onSelectItem?: (item: TItem) => void;
}) {
  const [addedItems, setAddedItems] = useState<TItem[]>([]);
  const options = useMemo(
    () => uniqueItems([...items, ...addedItems], itemToStringValue),
    [addedItems, itemToStringValue, items],
  );
  const selected = useMemo(
    () => options.find((item) => itemToStringValue(item) === value) ?? null,
    [itemToStringValue, options, value],
  );
  const newValue = value.trim();
  const addItem =
    newValue !== "" && !options.some((item) => itemToStringValue(item) === newValue)
      ? freeTextItem(newValue)
      : null;
  const renderedOptions = addItem ? [...options, addItem] : options;
  const hasRenderedOptions = renderedOptions.length > 0;

  return (
    <Combobox
      value={selected ? itemToStringValue(selected) : ""}
      inputValue={value}
      onInputValueChange={onChange}
      onValueChange={(next) => {
        if (!next) {
          return;
        }
        const item =
          renderedOptions.find((candidate) => itemToStringValue(candidate) === next) ??
          freeTextItem(next);
        const itemValue = itemToStringValue(item);

        if (!options.some((candidate) => itemToStringValue(candidate) === itemValue)) {
          setAddedItems((current) => uniqueItems([...current, item], itemToStringValue));
        }

        onChange(itemValue);
        onSelectItem?.(item);
      }}
      preserveInputOnBlur
    >
      <ComboboxAnchor className="w-full">
        <ComboboxInput
          id={id}
          name={name}
          placeholder={placeholder}
          disabled={disabled}
          aria-invalid={invalid}
          onBlur={onBlur}
        />
        {value !== "" ? <ComboboxCancel aria-label="Clear value" /> : null}
        <ComboboxTrigger aria-label="Open values" />
      </ComboboxAnchor>
      {hasRenderedOptions ? (
        <ComboboxContent>
          {options.map((item) => {
            const itemValue = itemToStringValue(item);
            return (
              <ComboboxItem
                key={itemKey?.(item) ?? itemValue}
                value={itemValue}
                label={itemValue}
                disabled={itemDisabled?.(item)}
              >
                {renderItem?.(item) ?? itemValue}
              </ComboboxItem>
            );
          })}
          {addItem ? (
            <>
              {options.length > 0 ? <ComboboxSeparator /> : null}
              <ComboboxItem value={newValue} label={newValue}>
                <span className="min-w-0 flex-1 truncate">Add &quot;{newValue}&quot;</span>
              </ComboboxItem>
            </>
          ) : null}
        </ComboboxContent>
      ) : null}
    </Combobox>
  );
}

function uniqueItems<TItem>(items: TItem[], itemToStringValue: (item: TItem) => string): TItem[] {
  const seen = new Set<string>();
  return items.filter((item) => {
    const value = itemToStringValue(item);
    if (seen.has(value)) {
      return false;
    }
    seen.add(value);
    return true;
  });
}
