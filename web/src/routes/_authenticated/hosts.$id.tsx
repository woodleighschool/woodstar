import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { hostQueryOptions } from "@/lib/queries/hosts";
import { parseRouteID } from "@/lib/route-params";
import { HostDetailPage } from "@/pages/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id")({
  staticData: { breadcrumb: HostBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(hostQueryOptions(parseRouteID(params.id)));
  },
  component: HostDetailPage,
});

function HostBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/hosts/$id" });
  const { data } = useQuery(hostQueryOptions(parseRouteID(id)));
  return data?.display_name ?? id;
}
