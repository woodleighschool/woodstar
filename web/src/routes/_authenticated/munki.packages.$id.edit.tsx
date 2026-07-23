import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { munkiPackageQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { MunkiPackageEditPage } from "@/pages/munki/packages/edit";

export const Route = createFileRoute("/_authenticated/munki/packages/$id/edit")({
  staticData: { breadcrumb: PackageBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(munkiPackageQueryOptions(parseRouteID(params.id)));
  },
  component: MunkiPackageEditPage,
});

function PackageBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/munki/packages/$id/edit" });
  const { data } = useQuery(munkiPackageQueryOptions(parseRouteID(id)));
  return data ? `${data.software.name} ${data.version}` : id;
}
