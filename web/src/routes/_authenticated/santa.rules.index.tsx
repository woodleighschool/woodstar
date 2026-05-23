import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SantaRulesPage } from "@/pages/santa/rules";

const searchSchema = z.object({
  q: z.string().optional(),
  page: z.coerce.number().int().min(1).optional(),
  per_page: z.coerce.number().int().min(10).max(200).optional(),
  order_key: z.string().optional(),
  order_direction: z.enum(["asc", "desc"]).optional(),
  rule_type: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/rules/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaRulesPage,
});
