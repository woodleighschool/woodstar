import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { SoftwareListPage } from "@/pages/software/list";

const sourceValues = [
  "apps",
  "homebrew_packages",
  "browser_plugins",
  "npm_packages",
  "ide_extensions",
  "go_binaries",
  "python_packages",
] as const;

const searchSchema = createListSearchSchema([
  "name",
  "source",
  "hosts_count",
  "versions_count",
]).extend({
  source: z.array(z.enum(sourceValues)).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/software/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: SoftwareListPage,
});
