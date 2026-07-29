import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { MunkiPackageListPage } from "@features/munki/packages/list";
import { MUNKI_INSTALLER_TYPE_VALUES } from "@features/munki/software/metadata";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "software_name",
  "software_display_name",
  "version",
  "type",
  "size",
  "updated_at",
]).extend({
  type: z.array(z.enum(MUNKI_INSTALLER_TYPE_VALUES)).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/packages/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: MunkiPackageListPage,
});
