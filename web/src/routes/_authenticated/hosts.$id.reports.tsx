import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { HostReportsPage } from "@features/hosts/detail";
import { REPORT_SNAPSHOT_STATUS_VALUES } from "@features/osquery/reports/query-results";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const SEARCH_DEFAULTS = {
  ...TABLE_SEARCH_DEFAULTS,
  sort: "reportName.asc",
} as const;

const searchSchema = createTableSearchSchema(["reportName", "status", "collectedAt", "rowCount"], {
  defaultSort: SEARCH_DEFAULTS.sort,
}).extend({
  status: z.enum(REPORT_SNAPSHOT_STATUS_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/hosts/$id/reports")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  component: HostReportsPage,
});
