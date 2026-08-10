import { Fragment, type ReactNode } from "react";
import { useMemo, useState } from "react";

import { InputGroupLoadingAddon } from "@components/input-group-loading-addon";
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
  loading?: boolean;
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
    loading = false,
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
  const [selectedItem, setSelectedItem] = useState<TItem | null>(null);
  const [open, setOpen] = useState(false);

  const itemToKey = itemKey ?? itemToStringValue;
  const retainedSelection =
    selectedItem && itemToStringValue(selectedItem) === value ? selectedItem : null;

  // Callers can keep option identity separate from the public string value.
  const options = useMemo(
    () =>
      uniqueItems(
        retainedSelection
          ? [...items, ...addedItems, retainedSelection]
          : [...items, ...addedItems],
        itemToKey,
      ),
    [addedItems, itemToKey, items, retainedSelection],
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

  const input = (
    <ComboboxInput
      id={id}
      name={name}
      className="w-full"
      placeholder={placeholder}
      disabled={disabled}
      aria-invalid={invalid}
      aria-busy={loading}
      onBlur={onBlur}
      showTrigger={mode === "create"}
      showClear={value !== ""}
    >
      {loading ? <InputGroupLoadingAddon /> : null}
    </ComboboxInput>
  );

  const content = (
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
  );

  return (
    <Combobox
      items={renderedOptions}
      filter={filterItems ? undefined : null}
      disabled={disabled}
      open={open && !loading && renderedOptions.length > 0}
      onOpenChange={setOpen}
      itemToStringLabel={itemToStringValue}
      itemToStringValue={itemToStringValue}
      isItemEqualToValue={(item, candidate) => itemToKey(item) === itemToKey(candidate)}
      value={selected}
      inputValue={value}
      onInputValueChange={(next, eventDetails) => {
        // Selection is handled by onValueChange so that onSelectItem receives
        // the selected TItem.
        if (
          eventDetails.reason === "input-change" ||
          eventDetails.reason === "input-clear" ||
          eventDetails.reason === "clear-press"
        ) {
          if (selectedItem && itemToStringValue(selectedItem) !== next) {
            setSelectedItem(null);
          }
          onChange(next);
          setOpen(next.trim() !== "");
        }
      }}
      onValueChange={(next) => {
        if (!next) {
          return;
        }

        const itemValue = itemToStringValue(next);
        setSelectedItem(next);

        if (addItem === next) {
          setAddedItems((current) => uniqueItems([...current, next], itemToStringValue));
        }

        onChange(itemValue);
        onSelectItem?.(next);
        setOpen(false);
      }}
    >
      {input}
      {content}
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
