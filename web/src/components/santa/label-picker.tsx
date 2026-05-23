import { Checkbox } from "@/components/ui/checkbox";
import { useLabels, type Label as WoodstarLabel } from "@/hooks/use-labels";

export function LabelPicker({ value, onChange }: { value: number[]; onChange: (value: number[]) => void }) {
  const labels = useLabels({ per_page: 500, order_key: "name", order_direction: "asc" });
  const rows = labels.data?.items ?? [];

  if (labels.isLoading) {
    return <p className="text-muted-foreground text-sm">Loading labels...</p>;
  }
  if (labels.error) {
    return <p className="text-destructive text-sm">{labels.error.message}</p>;
  }
  if (rows.length === 0) {
    return <p className="text-muted-foreground text-sm">No labels available.</p>;
  }

  return (
    <div className="grid max-h-48 gap-2 overflow-auto rounded-md border p-2">
      {rows.map((label) => (
        <LabelChoice
          key={label.id}
          label={label}
          selected={value.includes(label.id)}
          onChange={onChange}
          value={value}
        />
      ))}
    </div>
  );
}

function LabelChoice({
  label,
  selected,
  onChange,
  value,
}: {
  label: WoodstarLabel;
  selected: boolean;
  value: number[];
  onChange: (value: number[]) => void;
}) {
  return (
    <label className="hover:bg-muted/60 flex items-center gap-2 rounded-sm px-2 py-1 text-sm">
      <Checkbox
        checked={selected}
        onCheckedChange={(checked) => {
          if (checked) onChange([...value, label.id]);
          else onChange(value.filter((id) => id !== label.id));
        }}
      />
      <span className="truncate">{label.name}</span>
      <span className="text-muted-foreground ml-auto tabular-nums">{label.hosts_count}</span>
    </label>
  );
}
