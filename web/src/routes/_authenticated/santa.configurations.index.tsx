import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { ConfigurationListPage } from "@/pages/santa/configurations/list";

const searchSchema = createListSearchSchema(["name", "description", "position", "updated_at"]);

export const Route = createFileRoute("/_authenticated/santa/configurations/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: ConfigurationListPage,
});
