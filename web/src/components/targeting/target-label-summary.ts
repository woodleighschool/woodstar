import { targetSummary, type TargetSummaryInput } from "@/lib/targeting";

export function targetLabelSummary(targets: TargetSummaryInput) {
  return targetSummary(targets);
}
