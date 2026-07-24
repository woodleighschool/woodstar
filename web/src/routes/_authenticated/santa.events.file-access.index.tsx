import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { FILE_ACCESS_DECISION_VALUES } from "@/pages/santa/events/decisions";
import { SantaFileAccessEventListPage } from "@/pages/santa/events/list";

const searchSchema = createListSearchSchema([
  "occurred_at",
  "ingested_at",
  "decision",
  "host",
  "host_id",
  "rule_name",
  "target",
]).extend({
  decision: z.array(z.enum(FILE_ACCESS_DECISION_VALUES)).optional().catch(undefined),
  host_id: z.coerce.number().int().positive().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/file-access/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: SantaFileAccessEventListPage,
});
