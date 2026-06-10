import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { tableSearchSchema } from "@/lib/pagination";
import { DECISION_FILTER_VALUES } from "@/pages/santa/events/decisions";
import { SantaEventListPage } from "@/pages/santa/events/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  host_id: z.coerce.number().int().positive().optional(),
  user: z.string().optional(),
  decisions: z
    .preprocess(
      (value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value),
      z.array(z.enum(DECISION_FILTER_VALUES)),
    )
    .optional(),
});

export const Route = createFileRoute("/_authenticated/santa/events/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaEventListPage,
});
