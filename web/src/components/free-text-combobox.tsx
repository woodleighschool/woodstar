import { Fragment, type ReactNode } from "react";
import { useMemo, useState } from "react";

import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxSeparator,
} from "@components/ui/combobox";

type FreeTextComboboxCommonProps<TItem> = {
  id?: string;
  name?: string;
  value: string;
  items: TItem[];
  placeholder?: string;
  invalid?: boolean;
  disabled?: boolean;
  filterItems?: boolean;
  itemToStringValue: (item: TItem) => string;
  itemKey?: (item: TItem) => string;
  itemDisabled?: (item: TItem) => boolean;
  renderItem?: (item: TItem) => ReactNode;
  onBlur?: () => void;
  onChange: (value: string) => void;
  onSelectItem?: (item: TItem) => void;
};

type FreeTextComboboxProps<TItem> =
  | (FreeTextComboboxCommonProps<TItem> & {
      mode: "create";
      freeTextItem: (value: string) => TItem;
    })
  | (FreeTextComboboxCommonProps<TItem> & {
      mode: "free-text";
      freeTextItem?: never;
    });

export function FreeTextCombobox<TItem>(props: FreeTextComboboxProps<TItem>) {
  const {
    mode,
    id,
    name,
    value,
    items,
    placeholder,
    invalid,
    disabled,
    filterItems = true,
    itemToStringValue,
    itemKey,
    itemDisabled,
    renderItem,
    onBlur,
    onChange,
    onSelectItem,
  } = props;

  const [addedItems, setAddedItems] = useState<TItem[]>([]);

  const itemToKey = itemKey ?? itemToStringValue;

  // The public value is the string returned by itemToStringValue, so that
  // string must also define uniqueness within this component.
  const options = useMemo(
    () => uniqueItems([...items, ...addedItems], itemToStringValue),
    [addedItems, itemToStringValue, items],
  );

  const selected = options.find((item) => itemToStringValue(item) === value) ?? null;

  const newValue = value.trim();

  const addItem =
    mode === "create" &&
    newValue !== "" &&
    !options.some((item) => itemToStringValue(item) === newValue)
      ? props.freeTextItem(newValue)
      : null;

  const renderedOptions = addItem ? [...options, addItem] : options;

  return (
    <Combobox
      items={renderedOptions}
      filter={filterItems ? undefined : null}
      disabled={disabled}
      itemToStringLabel={itemToStringValue}
      itemToStringValue={itemToStringValue}
      isItemEqualToValue={(item, selectedItem) => itemToKey(item) === itemToKey(selectedItem)}
      value={mode === "create" ? selected : null}
      inputValue={value}
      onInputValueChange={(next, eventDetails) => {
        // Selection is handled by onValueChange so that onSelectItem receives
        // the selected TItem.
        if (eventDetails.reason !== "item-press") {
          onChange(next);
        }
      }}
      onValueChange={(next) => {
        if (!next) {
          return;
        }

        const itemValue = itemToStringValue(next);

        if (
          mode === "create" &&
          !options.some((candidate) => itemToStringValue(candidate) === itemValue)
        ) {
          setAddedItems((current) => uniqueItems([...current, next], itemToStringValue));
        }

        onChange(itemValue);
        onSelectItem?.(next);
      }}
    >
      <ComboboxInput
        id={id}
        name={name}
        className="w-full"
        placeholder={placeholder}
        disabled={disabled}
        aria-invalid={invalid}
        onBlur={onBlur}
        showClear={value !== ""}
      />

      {renderedOptions.length > 0 ? (
        <ComboboxContent>
          <ComboboxList>
            {(item: TItem, index: number) => {
              if (item === addItem) {
                return (
                  <Fragment key={`create:${newValue}`}>
                    {index > 0 ? <ComboboxSeparator /> : null}
                    <ComboboxItem value={item}>
                      <span className="min-w-0 flex-1 truncate">Add &quot;{newValue}&quot;</span>
                    </ComboboxItem>
                  </Fragment>
                );
              }

              const itemValue = itemToStringValue(item);

              return (
                <ComboboxItem key={itemToKey(item)} value={item} disabled={itemDisabled?.(item)}>
                  {renderItem?.(item) ?? itemValue}
                </ComboboxItem>
              );
            }}
          </ComboboxList>
        </ComboboxContent>
      ) : null}
    </Combobox>
  );
}

function uniqueItems<TItem>(items: TItem[], getValue: (item: TItem) => string): TItem[] {
  const seen = new Set<string>();

  return items.filter((item) => {
    const value = getValue(item);

    if (seen.has(value)) {
      return false;
    }

    seen.add(value);
    return true;
  });
}
