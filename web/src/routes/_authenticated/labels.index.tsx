import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { LabelListPage } from "@features/labels/list";
import { LABEL_MEMBERSHIP_VALUES } from "@features/labels/model";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@lib/list-search";

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
