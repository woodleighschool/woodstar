import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { munkiDistributionPointQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { DistributionPointDetailPage } from "@/pages/munki/distribution-points/detail";

export const Route = createFileRoute(
  "/_authenticated/munki/distribution-points/$distributionPointId",
)({
  staticData: { breadcrumb: DistributionPointBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      munkiDistributionPointQueryOptions(parseRouteID(params.distributionPointId)),
    );
  },
  component: DistributionPointDetailPage,
});

function DistributionPointBreadcrumb(): string {
  const { distributionPointId } = useParams({
    from: "/_authenticated/munki/distribution-points/$distributionPointId",
  });
  const { data } = useQuery(munkiDistributionPointQueryOptions(parseRouteID(distributionPointId)));
  return data?.name ?? distributionPointId;
}
