import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { MunkiSoftwareListPage } from "@features/munki/software/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "name",
  "display_name",
  "category",
  "developer",
  "updated_at",
]);

export const Route = createFileRoute("/_authenticated/munki/software/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: MunkiSoftwareListPage,
});
