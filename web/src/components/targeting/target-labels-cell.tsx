import type { ReactNode } from "react";

import { type LabelChip, labelsFromIDs } from "@/components/labels/label-chip-utils";
import { LabelChips } from "@/components/labels/label-chips";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { isAllHostsLabel } from "@/lib/labels";
import {
  type FlatLabelTarget,
  type LabelTargetSet,
  targetLabelIDs,
  targetSummary,
} from "@/lib/targeting";

export function TargetLabelsCell({
  labelIDs,
  targets,
  labelsByID,
  empty,
}: {
  labelIDs?: number[];
  targets?: LabelTargetSet | FlatLabelTarget[] | null;
  labelsByID: ReadonlyMap<number, LabelChip>;
  empty?: ReactNode;
}) {
  const ids = targets ? targetLabelIDs(targets) : (labelIDs ?? []);
  const countText = targets
    ? targetSummary(targets)
    : `${ids.length} label${ids.length === 1 ? "" : "s"}`;

  if (ids.length === 0) {
    return empty ?? <span className="text-sm tabular-nums">{countText}</span>;
  }

  const labels = labelsFromIDs(ids, labelsByID);
  if (!targets && labels.length === 1 && isAllHostsLabel(labels[0])) {
    return <span className="text-sm">{labels[0].name}</span>;
  }

  return (
    <HoverCard openDelay={150} closeDelay={150}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="rounded-sm text-sm tabular-nums underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
        >
          {countText}
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" side="top" className="w-auto max-w-80 p-2">
        <LabelChips labels={labels} />
      </HoverCardContent>
    </HoverCard>
  );
}
