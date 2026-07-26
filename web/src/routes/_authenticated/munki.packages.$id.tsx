import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { munkiPackageQueryOptions } from "@features/munki/packages/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/packages/$id")({
  staticData: { breadcrumb: PackageBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(munkiPackageQueryOptions(parseRouteID(params.id)));
  },
});

function PackageBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/munki/packages/$id" });
  const { data } = useQuery(munkiPackageQueryOptions(parseRouteID(id)));
  return data ? `${data.software.name} ${data.version}` : id;
}
