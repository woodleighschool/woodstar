import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { MunkiSoftwareDetailPage } from "@features/munki/software/detail";
import { SOFTWARE_REPORT_STATUS_VALUES } from "@features/munki/software/report";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const SEARCH_DEFAULTS = {
  ...TABLE_SEARCH_DEFAULTS,
  sort: "host_name.asc",
} as const;

const searchSchema = createTableSearchSchema(
  ["host_name", "target_version", "status", "evaluated_at"],
  { defaultSort: SEARCH_DEFAULTS.sort },
).extend({
  status: z.array(z.enum(SOFTWARE_REPORT_STATUS_VALUES)).optional().catch(undefined),
  tab: z.literal("report").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/software/$id/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  component: MunkiSoftwareDetailPage,
});
