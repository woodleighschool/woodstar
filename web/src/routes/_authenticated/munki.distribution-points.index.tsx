import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { DistributionPointListPage } from "@features/munki/distribution-points/list";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@lib/list-search";

const searchSchema = createListSearchSchema(["name", "position"]);

export const Route = createFileRoute("/_authenticated/munki/distribution-points/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: DistributionPointListPage,
});
