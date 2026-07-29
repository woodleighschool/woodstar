import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { DistributionPointListPage } from "@features/munki/distribution-points/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "position"]);

export const Route = createFileRoute("/_authenticated/munki/distribution-points/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: DistributionPointListPage,
});
