import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { MUNKI_ASSIGNMENT_ACTION_VALUES } from "@features/munki/software/actions";
import { MUNKI_DEPLOYMENT_STATUS_VALUES } from "@features/munki/software/deployment";
import { MunkiSoftwareDetailPage } from "@features/munki/software/detail";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const hostTableSearchSchema = createTableSearchSchema([
  "display_name",
  "status",
  "installed_version",
  "target_version",
  "last_successful_at",
]);

const SEARCH_DEFAULTS = {
  host_page: TABLE_SEARCH_DEFAULTS.page,
  host_per_page: TABLE_SEARCH_DEFAULTS.per_page,
} as const;

const searchSchema = z.object({
  tab: z.enum(["targets", "packages"]).optional().catch(undefined),
  host_q: hostTableSearchSchema.shape.q,
  host_page: hostTableSearchSchema.shape.page,
  host_per_page: hostTableSearchSchema.shape.per_page,
  host_sort: hostTableSearchSchema.shape.sort,
  status: z.enum(MUNKI_DEPLOYMENT_STATUS_VALUES).optional().catch(undefined),
  action: z.enum(MUNKI_ASSIGNMENT_ACTION_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/software/$id/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  component: MunkiSoftwareDetailPage,
});
