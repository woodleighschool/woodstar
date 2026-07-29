import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { CheckListPage } from "@features/osquery/checks/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "created_at", "updated_at"]);

export const Route = createFileRoute("/_authenticated/osquery/checks/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: CheckListPage,
});
