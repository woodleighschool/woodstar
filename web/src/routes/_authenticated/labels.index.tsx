import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { LABEL_MEMBERSHIP_VALUES } from "@/lib/labels";
import { tableSearchSchema } from "@/lib/pagination";
import { LabelListPage } from "@/pages/labels/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  label_membership_type: z.enum(LABEL_MEMBERSHIP_VALUES).optional(),
});

export const Route = createFileRoute("/_authenticated/labels/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: LabelListPage,
});
