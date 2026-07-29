import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { CheckDetailPage } from "@features/osquery/checks/detail";
import { CHECK_RESULT_STATUS_VALUES } from "@features/osquery/checks/model";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const SEARCH_DEFAULTS = {
  ...TABLE_SEARCH_DEFAULTS,
  sort: "host_name.asc",
} as const;

const searchSchema = createTableSearchSchema(["host_name", "status", "updated_at"], {
  defaultSort: SEARCH_DEFAULTS.sort,
}).extend({
  status: z.enum(CHECK_RESULT_STATUS_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/checks/$id/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  component: CheckDetailPage,
});
