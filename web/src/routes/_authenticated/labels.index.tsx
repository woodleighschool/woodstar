import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { LabelsPage } from "@/pages/labels/list";

const searchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(200).optional(),
  sort: z.string().optional(),
  label_membership_type: z.string().optional(),
  platform: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/labels/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: LabelsPage,
});
