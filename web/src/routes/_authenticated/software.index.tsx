import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { SoftwareListPage } from "@features/software/list";
import { SOFTWARE_SOURCE_FILTER_VALUES } from "@features/software/software-source-labels";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "source", "hosts_count", "versions"]).extend({
  source: z.array(z.enum(SOFTWARE_SOURCE_FILTER_VALUES)).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/software/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: SoftwareListPage,
});
