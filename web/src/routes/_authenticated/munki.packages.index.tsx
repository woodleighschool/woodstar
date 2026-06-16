import { createFileRoute } from "@tanstack/react-router";

import { MunkiPackageListPage } from "@/pages/munki/packages/list";

// Pure list route: q, page, per_page, sort, and the type facet are nuqs-owned.
export const Route = createFileRoute("/_authenticated/munki/packages/")({
  component: MunkiPackageListPage,
});
