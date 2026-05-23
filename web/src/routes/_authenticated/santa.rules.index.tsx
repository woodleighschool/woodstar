import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { SantaRulesPage } from "@/pages/santa/rules";

const searchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
  rule_type: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/rules/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaRulesPage,
});
