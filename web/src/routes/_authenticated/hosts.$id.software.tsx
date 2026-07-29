import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { HostSoftwarePage } from "@features/hosts/detail";
import { SOFTWARE_SOURCE_FILTER_VALUES } from "@features/software/software-source-labels";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "name",
  "version",
  "source",
  "last_opened_at",
]).extend({
  source: z.array(z.enum(SOFTWARE_SOURCE_FILTER_VALUES)).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/hosts/$id/software")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: HostSoftwarePage,
});
