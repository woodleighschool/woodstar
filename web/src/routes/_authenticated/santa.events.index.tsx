import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { DECISION_FILTER_VALUES } from "@features/santa/events/decisions";
import { SantaEventListPage } from "@features/santa/events/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
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
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: SantaEventListPage,
});
