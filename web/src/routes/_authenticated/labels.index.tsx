import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { LABEL_MEMBERSHIP_VALUES } from "@/lib/labels";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { LabelListPage } from "@/pages/labels/list";

const searchSchema = createListSearchSchema([
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
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: LabelListPage,
});
