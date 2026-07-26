import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { MunkiSoftwareEditPage } from "@features/munki/software/edit";
import { munkiSoftwareQueryOptions } from "@features/munki/software/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/software/$id")({
  staticData: { breadcrumb: SoftwareBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(munkiSoftwareQueryOptions(parseRouteID(params.id)));
  },
  component: MunkiSoftwareEditPage,
});

function SoftwareBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/munki/software/$id" });
  const { data } = useQuery(munkiSoftwareQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
