import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { RULE_TYPE_VALUES } from "@/lib/santa-rules";
import { RuleCreatePage } from "@/pages/santa/rules/create";

const searchSchema = z.object({
  rule_type: z.enum(RULE_TYPE_VALUES).optional().catch(undefined),
  identifier: z.string().optional().catch(undefined),
  name: z.string().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/rules/new")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  component: RuleCreatePage,
});
