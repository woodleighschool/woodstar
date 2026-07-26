import { z } from "zod";

import type { LabelRef } from "@lib/api";

export type LabelTargetSet = {
  include: LabelRef[];
  exclude: LabelRef[];
};

const labelRefSchema = z.object({
  label_id: z.number().int("Label selection is invalid.").positive("Select a label."),
});

// Validates the include/exclude label sets shared by checks, reports, and santa
// configurations. Their target editors only emit real label ids, so this mainly
// lets the form-level schema cover the full mutation shape.
export const labelTargetSetSchema = z.object({
  include: z.array(labelRefSchema),
  exclude: z.array(labelRefSchema),
});

export function emptyLabelTargetSet(): LabelTargetSet {
  return { include: [], exclude: [] };
}

export function normalizeLabelTargetSet(
  targets: LabelTargetSet | null | undefined,
): LabelTargetSet {
  return {
    include: targets?.include ?? [],
    exclude: targets?.exclude ?? [],
  };
}
