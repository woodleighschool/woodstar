import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { PolicyListPage } from "@features/osquery/policies/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "created_at", "updated_at"]);

export const Route = createFileRoute("/_authenticated/osquery/policies/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: PolicyListPage,
});
