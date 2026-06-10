import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { tableSearchSchema } from "@/lib/pagination";
import { FILE_ACCESS_DECISION_VALUES } from "@/pages/santa/events/decisions";
import { SantaFileAccessEventListPage } from "@/pages/santa/events/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  host_id: z.coerce.number().int().positive().optional(),
  decisions: z
    .preprocess(
      (value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value),
      z.array(z.enum(FILE_ACCESS_DECISION_VALUES)),
    )
    .optional(),
});

export const Route = createFileRoute("/_authenticated/santa/events/file-access/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaFileAccessEventListPage,
});
