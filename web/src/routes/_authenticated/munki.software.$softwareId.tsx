import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { munkiSoftwareQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { MunkiSoftwareEditPage } from "@/pages/munki/software/edit";

export const Route = createFileRoute("/_authenticated/munki/software/$softwareId")({
  staticData: { breadcrumb: SoftwareBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      munkiSoftwareQueryOptions(parseRouteID(params.softwareId)),
    );
  },
  component: MunkiSoftwareEditPage,
});

function SoftwareBreadcrumb(): string {
  const { softwareId } = useParams({ from: "/_authenticated/munki/software/$softwareId" });
  const { data } = useQuery(munkiSoftwareQueryOptions(parseRouteID(softwareId)));
  return data?.name ?? softwareId;
}
