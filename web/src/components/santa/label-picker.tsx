import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
} from "@/components/ui/combobox";
import { useLabels, type Label as WoodstarLabel } from "@/hooks/use-labels";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

interface LabelPickerProps {
  value: number[];
  onChange: (value: number[]) => void;
  unavailableLabelIDs?: readonly number[];
  emptyMessage?: string;
  emptyPlaceholder?: string;
}

export function LabelPicker({
  value,
  onChange,
  unavailableLabelIDs = [],
  emptyMessage,
  emptyPlaceholder,
}: LabelPickerProps) {
  const labels = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
    label_type: "regular",
  });
  const rows = labels.data?.items ?? [];
  const unavailable = new Set(unavailableLabelIDs);
  const items = rows.filter((label) => value.includes(label.id) || !unavailable.has(label.id));
  const selected = rows.filter((label) => value.includes(label.id));
  const noLabelsMessage = emptyMessage ?? "No labels available.";

  if (labels.isLoading) {
    return <p className="text-muted-foreground text-sm">Loading labels...</p>;
  }
  if (labels.error) {
    return <p className="text-destructive text-sm">{labels.error.message}</p>;
  }

  return (
    <Combobox
      items={items}
      multiple
      value={selected}
      itemToStringValue={(label) => label.name}
      onValueChange={(next) => onChange(next.map((label) => label.id))}
    >
      <ComboboxChips>
        <ComboboxValue>
          {selected.map((label) => (
            <ComboboxChip key={label.id}>{label.name}</ComboboxChip>
          ))}
        </ComboboxValue>
        <ComboboxChipsInput
          placeholder={items.length === 0 ? (emptyPlaceholder ?? "No labels available") : "Add label"}
        />
      </ComboboxChips>
      <ComboboxContent>
        <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No labels found."}</ComboboxEmpty>
        <ComboboxList>
          {(label: WoodstarLabel) => (
            <ComboboxItem key={label.id} value={label}>
              <span className="min-w-0 flex-1 truncate">{label.name}</span>
              <span className="text-muted-foreground tabular-nums">{label.hosts_count}</span>
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
