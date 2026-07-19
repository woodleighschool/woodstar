import { createFileRoute } from "@tanstack/react-router";

import { DistributionPointCreatePage } from "@/pages/munki/distribution-points/create";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/new")({
  staticData: { breadcrumb: "New" },
  component: DistributionPointCreatePage,
});
