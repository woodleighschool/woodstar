import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { tableSearchSchema } from "@/lib/pagination";
import { RULE_TYPE_VALUES } from "@/lib/santa-rules";
import { RuleListPage } from "@/pages/santa/rules/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  rule_type: z.enum(RULE_TYPE_VALUES).optional(),
});

export const Route = createFileRoute("/_authenticated/santa/rules/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: RuleListPage,
});
