import { createFileRoute } from "@tanstack/react-router";

import { DistributionPointDetailPage } from "@/pages/munki/distribution-points/detail";

export const Route = createFileRoute(
  "/_authenticated/munki/distribution-points/$distributionPointId",
)({
  component: DistributionPointDetailPage,
});
