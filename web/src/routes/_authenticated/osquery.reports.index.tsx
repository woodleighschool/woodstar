import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { ReportListPage } from "@/pages/osquery/reports/list";

const searchSchema = createListSearchSchema([
  "name",
  "created_at",
  "updated_at",
  "schedule_interval",
]);

export const Route = createFileRoute("/_authenticated/osquery/reports/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: ReportListPage,
});
