import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { MUNKI_SOFTWARE_ACTION_VALUES } from "@features/munki/software/actions";
import {
  INSTALLATION_STATUS_VALUES,
  MUNKI_RESULT_VALUES,
} from "@features/munki/software/deployment";
import { MunkiSoftwareDetailPage } from "@features/munki/software/detail";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const hostTableSearchSchema = createTableSearchSchema([
  "display_name",
  "status",
  "installed_version",
  "munki_result",
  "target_version",
  "last_collected_at",
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
  status: z.enum(INSTALLATION_STATUS_VALUES).optional().catch(undefined),
  munki_result: z.enum(MUNKI_RESULT_VALUES).optional().catch(undefined),
  action: z.enum(MUNKI_SOFTWARE_ACTION_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/software/$id/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  component: MunkiSoftwareDetailPage,
});
