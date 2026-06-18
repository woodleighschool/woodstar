import { createFileRoute } from "@tanstack/react-router";

import { DistributionPointListPage } from "@/pages/munki/distribution-points/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned.
export const Route = createFileRoute("/_authenticated/munki/distribution-points/")({
  component: DistributionPointListPage,
});
