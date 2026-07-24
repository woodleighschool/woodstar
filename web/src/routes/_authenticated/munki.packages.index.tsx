import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { MunkiPackageListPage } from "@/pages/munki/packages/list";
import { MUNKI_INSTALLER_TYPE_VALUES } from "@/pages/munki/software/munki-software";

const searchSchema = createListSearchSchema([
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
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: MunkiPackageListPage,
});
