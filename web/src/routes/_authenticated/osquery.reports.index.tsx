import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { ReportListPage } from "@features/osquery/reports/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "name",
  "created_at",
  "updated_at",
  "schedule_interval",
]);

export const Route = createFileRoute("/_authenticated/osquery/reports/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: ReportListPage,
});
