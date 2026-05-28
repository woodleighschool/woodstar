import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
} from "@/components/ui/combobox";
import { useLabels, type Label as WoodstarLabel } from "@/hooks/use-labels";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

interface LabelPickerProps {
  value: number[];
  onChange: (value: number[]) => void;
  selectionMode?: "multiple" | "single";
  includeBuiltins?: boolean;
  unavailableLabelIDs?: readonly number[];
  emptyMessage?: string;
  emptyPlaceholder?: string;
  placeholder?: string;
  required?: boolean;
}

export function LabelPicker({
  value,
  onChange,
  selectionMode = "multiple",
  includeBuiltins = false,
  unavailableLabelIDs = [],
  emptyMessage,
  emptyPlaceholder,
  placeholder = "Add Label",
  required = false,
}: LabelPickerProps) {
  const labels = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
    label_type: includeBuiltins ? undefined : "regular",
  });
  const rows = labels.data?.items ?? [];
  const unavailable = new Set(unavailableLabelIDs);
  const items = rows.filter(
    (label) =>
      (includeBuiltins || label.label_type === "regular") && (value.includes(label.id) || !unavailable.has(label.id)),
  );
  const selected = rows.filter((label) => value.includes(label.id));
  const noLabelsMessage = emptyMessage ?? "No Labels Available.";

  if (labels.isLoading) {
    return <p className="text-muted-foreground text-sm">Loading Labels...</p>;
  }
  if (labels.error) {
    return <p className="text-destructive text-sm">{labels.error.message}</p>;
  }

  if (selectionMode === "single") {
    return (
      <Combobox
        items={items}
        value={selected[0] ?? null}
        itemToStringLabel={(label) => label.name}
        itemToStringValue={(label) => String(label.id)}
        onValueChange={(next) => onChange(next ? [next.id] : [])}
      >
        <ComboboxInput
          placeholder={items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder}
          required={required}
          showClear
        />
        <ComboboxContent>
          <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
          <ComboboxList>{labelItem}</ComboboxList>
        </ComboboxContent>
      </Combobox>
    );
  }

  return (
    <Combobox
      items={items}
      multiple
      value={selected}
      itemToStringLabel={(label) => label.name}
      itemToStringValue={(label) => String(label.id)}
      onValueChange={(next) => onChange(next.map((label) => label.id))}
    >
      <ComboboxChips>
        <ComboboxValue>
          {selected.map((label) => (
            <ComboboxChip key={label.id}>{label.name}</ComboboxChip>
          ))}
        </ComboboxValue>
        <ComboboxChipsInput
          placeholder={items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder}
          required={required && selected.length === 0}
        />
      </ComboboxChips>
      <ComboboxContent>
        <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
        <ComboboxList>{labelItem}</ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

function labelItem(label: WoodstarLabel) {
  return (
    <ComboboxItem key={label.id} value={label}>
      <span className="min-w-0 flex-1 truncate">{label.name}</span>
      <span className="text-muted-foreground tabular-nums">{label.hosts_count}</span>
    </ComboboxItem>
  );
}
