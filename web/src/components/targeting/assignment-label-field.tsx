import { LabelPicker } from "@/components/labels/label-picker";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function AssignmentLabelField({
  value,
  onChange,
  unavailableLabelIDs,
  includeBuiltins = true,
}: {
  value: number | null;
  onChange: (labelID: number | null) => void;
  unavailableLabelIDs: readonly number[];
  includeBuiltins?: boolean;
}) {
  return (
    <Field>
      <FieldLabel required>Label</FieldLabel>
      <LabelPicker
        value={value === null ? [] : [value]}
        onChange={(ids) => onChange(ids[0] ?? null)}
        selectionMode="single"
        includeBuiltins={includeBuiltins}
        unavailableLabelIDs={unavailableLabelIDs}
        required
        placeholder="Select Label"
      />
    </Field>
  );
}

// Edit dialogs only change metadata; the label is fixed once the assignment exists.
export function AssignmentLabelStatic({ name }: { name: string }) {
  return (
    <Field>
      <FieldLabel>Label</FieldLabel>
      <Input value={name} disabled readOnly />
    </Field>
  );
}
