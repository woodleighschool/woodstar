import type { ReactNode } from "react";
import { useMemo } from "react";

import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";

export function FreeTextCombobox<TItem>({
  id,
  name,
  value,
  items,
  placeholder,
  invalid,
  disabled,
  emptyText,
  noResultsText = "No Values Found.",
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
  emptyText?: ReactNode;
  noResultsText?: ReactNode;
  itemToStringValue: (item: TItem) => string;
  freeTextItem: (value: string) => TItem;
  itemKey?: (item: TItem) => string;
  itemDisabled?: (item: TItem) => boolean;
  renderItem?: (item: TItem) => ReactNode;
  onBlur?: () => void;
  onChange: (value: string) => void;
  onSelectItem?: (item: TItem) => void;
}) {
  const selected = useMemo(
    () =>
      items.find((item) => itemToStringValue(item) === value) ??
      (value ? freeTextItem(value) : null),
    [freeTextItem, itemToStringValue, items, value],
  );

  return (
    <Combobox
      items={items}
      value={selected}
      inputValue={value}
      itemToStringLabel={itemToStringValue}
      itemToStringValue={itemToStringValue}
      onInputValueChange={onChange}
      onValueChange={(next) => {
        if (!next) {
          return;
        }
        onChange(itemToStringValue(next));
        onSelectItem?.(next);
      }}
    >
      <ComboboxInput
        id={id}
        name={name}
        className="w-full"
        placeholder={placeholder}
        showClear={value !== ""}
        disabled={disabled}
        aria-invalid={invalid}
        onBlur={onBlur}
      />
      <ComboboxContent>
        <ComboboxEmpty>{items.length === 0 ? emptyText : noResultsText}</ComboboxEmpty>
        <ComboboxList>
          {(item: TItem) => (
            <ComboboxItem
              key={itemKey?.(item) ?? itemToStringValue(item)}
              value={item}
              disabled={itemDisabled?.(item)}
            >
              {renderItem?.(item) ?? itemToStringValue(item)}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
