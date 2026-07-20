import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { munkiPackageQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { MunkiPackageEditPage } from "@/pages/munki/packages/edit";

export const Route = createFileRoute("/_authenticated/munki/packages/$packageId/edit")({
  staticData: { breadcrumb: PackageBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      munkiPackageQueryOptions(parseRouteID(params.packageId)),
    );
  },
  component: MunkiPackageEditPage,
});

function PackageBreadcrumb(): string {
  const { packageId } = useParams({ from: "/_authenticated/munki/packages/$packageId/edit" });
  const { data } = useQuery(munkiPackageQueryOptions(parseRouteID(packageId)));
  return data ? `${data.software.name} ${data.version}` : packageId;
}
