import type { ReactNode } from "react";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { LabelRefList } from "@features/labels/components/label-ref-list";
import type { LabelRef } from "@lib/api";

export function TargetDetails({
  include,
  excludeLabelIDs,
}: {
  include: ReactNode;
  excludeLabelIDs: readonly number[];
}) {
  return (
    <KeyValueSection title="Targets">
      <KeyValueRow label="Include" value={include} />
      <KeyValueRow label="Exclude" value={<LabelRefList labelIDs={excludeLabelIDs} />} />
    </KeyValueSection>
  );
}

export function LabelTargetDetails({
  targets,
}: {
  targets: {
    include: readonly LabelRef[];
    exclude: readonly LabelRef[];
  };
}) {
  return (
    <TargetDetails
      include={<LabelRefList labelIDs={targets.include.map((target) => target.label_id)} />}
      excludeLabelIDs={targets.exclude.map((target) => target.label_id)}
    />
  );
}
