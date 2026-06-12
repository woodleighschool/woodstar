import { LabelAssignmentList } from "@/components/targeting/label-assignment-list";
import { Separator } from "@/components/ui/separator";
import { type LabelTargetSet, normalizeLabelTargetSet } from "@/lib/targeting";

export function LabelTargetSetEditor({
  value,
  onChange,
  includeBuiltins = true,
}: {
  value: LabelTargetSet;
  onChange: (next: LabelTargetSet) => void;
  includeBuiltins?: boolean;
}) {
  const targetSet = normalizeLabelTargetSet(value);
  const includeLabelIDs = targetSet.include.map((row) => row.label_id);
  const excludeLabelIDs = targetSet.exclude.map((row) => row.label_id);

  return (
    <div className="flex flex-col gap-6">
      <LabelAssignmentList
        title="Include"
        addLabel="Add Include"
        emptyText="No includes yet"
        rows={targetSet.include}
        crossListLabelIDs={excludeLabelIDs}
        includeBuiltins={includeBuiltins}
        onChange={(include) => onChange({ ...targetSet, include })}
      />
      <Separator />
      <LabelAssignmentList
        title="Exclude"
        addLabel="Add Exclude"
        emptyText="No excludes yet"
        rows={targetSet.exclude}
        crossListLabelIDs={includeLabelIDs}
        includeBuiltins={includeBuiltins}
        onChange={(exclude) => onChange({ ...targetSet, exclude })}
      />
    </div>
  );
}
