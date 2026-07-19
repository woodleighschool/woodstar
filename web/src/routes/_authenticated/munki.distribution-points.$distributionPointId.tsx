import { createFileRoute } from "@tanstack/react-router";

import { munkiDistributionPointQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { DistributionPointDetailPage } from "@/pages/munki/distribution-points/detail";

export const Route = createFileRoute(
  "/_authenticated/munki/distribution-points/$distributionPointId",
)({
  loader: async ({ context, params }) => {
    const distributionPoint = await context.queryClient.ensureQueryData(
      munkiDistributionPointQueryOptions(parseRouteID(params.distributionPointId)),
    );
    return { breadcrumb: distributionPoint.name };
  },
  component: DistributionPointDetailPage,
});
