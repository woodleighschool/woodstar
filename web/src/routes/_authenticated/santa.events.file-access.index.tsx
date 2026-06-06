import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { FILE_ACCESS_DECISION_VALUES } from "@/lib/santa-events";
import { SantaFileAccessEventsPage } from "@/pages/santa/events/list";

const searchSchema = z.object({
  q: z.string().optional(),
  host_id: z.coerce.number().int().positive().optional(),
  decisions: z
    .preprocess(
      (value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value),
      z.array(z.enum(FILE_ACCESS_DECISION_VALUES)),
    )
    .optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/events/file-access/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaFileAccessEventsPage,
});
