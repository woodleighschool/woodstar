import type { ReactNode } from "react";

import { labelsFromIDs, type LabelChip } from "@/components/labels/label-chip-utils";
import { LabelChips } from "@/components/labels/label-chips";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";

export function TargetLabelsCell({
  labelIDs,
  labelsByID,
  empty,
}: {
  labelIDs: number[];
  labelsByID: ReadonlyMap<number, LabelChip>;
  empty?: ReactNode;
}) {
  const countText = `${labelIDs.length} label${labelIDs.length === 1 ? "" : "s"}`;

  if (labelIDs.length === 0) {
    return empty ?? <span className="text-sm tabular-nums">{countText}</span>;
  }

  const labels = labelsFromIDs(labelIDs, labelsByID);
  if (labels.length === 1 && labels[0].name === "All Hosts") {
    return <span className="text-sm">All Hosts</span>;
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
