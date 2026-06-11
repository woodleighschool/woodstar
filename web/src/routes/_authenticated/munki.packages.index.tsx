import { createFileRoute } from "@tanstack/react-router";

import { MunkiPackageListPage } from "@/pages/munki/packages/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned.
export const Route = createFileRoute("/_authenticated/munki/packages/")({
  component: MunkiPackageListPage,
});
