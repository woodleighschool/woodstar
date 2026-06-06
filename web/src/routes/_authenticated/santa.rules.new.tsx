import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SantaRuleResourcePage } from "@/pages/santa/rules/form";
import { RULE_TYPE_VALUES } from "@/pages/santa/rules/shared";

const searchSchema = z.object({
  rule_type: z.enum(RULE_TYPE_VALUES).optional(),
  identifier: z.string().optional(),
  name: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/rules/new")({
  validateSearch: (search) => searchSchema.parse(search),
  component: () => <SantaRuleResourcePage mode="create" />,
});
