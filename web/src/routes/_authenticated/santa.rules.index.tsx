import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SantaRulesPage } from "@/pages/santa/rules";

const searchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(200).optional(),
  sort: z.string().optional(),
  rule_type: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/rules/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaRulesPage,
});
