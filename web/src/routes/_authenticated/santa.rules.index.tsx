import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { RULE_TYPE_VALUES } from "@/lib/santa-rules";
import { RuleListPage } from "@/pages/santa/rules/list";

const searchSchema = createListSearchSchema([
  "rule_type",
  "identifier",
  "name",
  "description",
  "updated_at",
]).extend({
  rule_type: z.enum(RULE_TYPE_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/rules/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: RuleListPage,
});
