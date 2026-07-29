import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { ReportDetailPage } from "@features/osquery/reports/detail";
import { REPORT_SNAPSHOT_STATUS_VALUES } from "@features/osquery/reports/query-results";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const SEARCH_DEFAULTS = {
  ...TABLE_SEARCH_DEFAULTS,
  sort: "hostName.asc",
} as const;

const searchSchema = createTableSearchSchema(["hostName", "status", "collectedAt", "rowCount"], {
  defaultSort: SEARCH_DEFAULTS.sort,
}).extend({
  status: z.enum(REPORT_SNAPSHOT_STATUS_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  component: ReportDetailPage,
});
