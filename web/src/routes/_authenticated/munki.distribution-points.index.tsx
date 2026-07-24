import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { DistributionPointListPage } from "@/pages/munki/distribution-points/list";

const searchSchema = createListSearchSchema(["name", "position"]);

export const Route = createFileRoute("/_authenticated/munki/distribution-points/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: DistributionPointListPage,
});
