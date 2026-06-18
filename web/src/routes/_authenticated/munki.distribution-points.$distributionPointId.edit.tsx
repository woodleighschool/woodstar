import { createFileRoute } from "@tanstack/react-router";

import { DistributionPointEditPage } from "@/pages/munki/distribution-points/edit";

export const Route = createFileRoute(
  "/_authenticated/munki/distribution-points/$distributionPointId/edit",
)({
  component: DistributionPointEditPage,
});
