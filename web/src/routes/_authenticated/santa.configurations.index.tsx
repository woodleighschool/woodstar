import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { ConfigurationListPage } from "@features/santa/configurations/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "description", "position", "updated_at"]);

export const Route = createFileRoute("/_authenticated/santa/configurations/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: ConfigurationListPage,
});
