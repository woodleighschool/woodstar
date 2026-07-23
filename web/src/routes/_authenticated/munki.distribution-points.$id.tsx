import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { munkiDistributionPointQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/$id")({
  staticData: { breadcrumb: DistributionPointBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      munkiDistributionPointQueryOptions(parseRouteID(params.id)),
    );
  },
});

function DistributionPointBreadcrumb(): string {
  const { id } = useParams({
    from: "/_authenticated/munki/distribution-points/$id",
  });
  const { data } = useQuery(munkiDistributionPointQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
