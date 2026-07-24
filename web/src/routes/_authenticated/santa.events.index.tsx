import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { DECISION_FILTER_VALUES } from "@/pages/santa/events/decisions";
import { SantaEventListPage } from "@/pages/santa/events/list";

const searchSchema = createListSearchSchema([
  "occurred_at",
  "ingested_at",
  "decision",
  "host",
  "host_id",
  "executing_user",
  "file_name",
]).extend({
  decision: z.array(z.enum(DECISION_FILTER_VALUES)).optional().catch(undefined),
  host_id: z.coerce.number().int().positive().optional().catch(undefined),
  user: z.string().trim().min(1).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: SantaEventListPage,
});
