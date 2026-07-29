import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { LabelListPage } from "@features/labels/list";
import { LABEL_MEMBERSHIP_VALUES } from "@features/labels/model";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "name",
  "label_type",
  "label_membership_type",
  "hosts_count",
  "updated_at",
]).extend({
  label_membership_type: z.enum(LABEL_MEMBERSHIP_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/labels/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: LabelListPage,
});
