import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { munkiDistributionPointQueryOptions } from "@features/munki/distribution-points/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/$id")({
  staticData: {
    breadcrumb: resourceName(
      munkiDistributionPointQueryOptions,
      (distributionPoint) => distributionPoint.name,
    ),
  },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      munkiDistributionPointQueryOptions(parseRouteID(params.id)),
    );
  },
});
