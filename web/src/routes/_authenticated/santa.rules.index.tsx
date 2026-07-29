import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { RuleListPage } from "@features/santa/rules/list";
import { RULE_TYPE_VALUES } from "@features/santa/rules/metadata";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
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
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: RuleListPage,
});
