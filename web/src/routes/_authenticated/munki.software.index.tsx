import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { MunkiSoftwareListPage } from "@features/munki/software/list";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@lib/list-search";

const searchSchema = createListSearchSchema([
  "name",
  "display_name",
  "category",
  "developer",
  "updated_at",
]);

export const Route = createFileRoute("/_authenticated/munki/software/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: MunkiSoftwareListPage,
});
