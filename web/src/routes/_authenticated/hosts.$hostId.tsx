import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { hostQueryOptions } from "@/lib/queries/hosts";
import { parseRouteID } from "@/lib/route-params";
import { HostDetailPage } from "@/pages/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$hostId")({
  staticData: { breadcrumb: HostBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(hostQueryOptions(parseRouteID(params.hostId)));
  },
  component: HostDetailPage,
});

function HostBreadcrumb(): string {
  const { hostId } = useParams({ from: "/_authenticated/hosts/$hostId" });
  const { data } = useQuery(hostQueryOptions(parseRouteID(hostId)));
  return data?.display_name ?? hostId;
}
