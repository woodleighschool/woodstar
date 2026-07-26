import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { CheckListPage } from "@features/osquery/checks/list";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@lib/list-search";

const searchSchema = createListSearchSchema(["name", "created_at", "updated_at"]);

export const Route = createFileRoute("/_authenticated/osquery/checks/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: CheckListPage,
});
