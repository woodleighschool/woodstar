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

export function LabelPicker({ value, onChange }: { value: number[]; onChange: (value: number[]) => void }) {
  const labels = useLabels({
    per_page: 500,
    order_key: "name",
    order_direction: "asc",
    label_type: "regular",
    platform: "darwin",
  });
  const rows = labels.data?.items ?? [];
  const selected = rows.filter((label) => value.includes(label.id));

  if (labels.isLoading) {
    return <p className="text-muted-foreground text-sm">Loading labels...</p>;
  }
  if (labels.error) {
    return <p className="text-destructive text-sm">{labels.error.message}</p>;
  }

  return (
    <Combobox
      items={rows}
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
        <ComboboxChipsInput placeholder={rows.length === 0 ? "No macOS labels available" : "Add label"} />
      </ComboboxChips>
      <ComboboxContent>
        <ComboboxEmpty>{rows.length === 0 ? "No macOS labels available." : "No labels found."}</ComboboxEmpty>
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
