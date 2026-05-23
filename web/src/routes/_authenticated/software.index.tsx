import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { SoftwarePage } from "@/pages/software/list";

const searchSchema = z.object({
  q: z.string().optional(),
  source: z
    .preprocess((value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value), z.array(z.string()))
    .optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/software/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SoftwarePage,
});
