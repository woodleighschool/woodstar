import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { softwareTitleQueryOptions } from "@/lib/queries/software";
import { parseRouteID } from "@/lib/route-params";
import { SoftwareDetailPage } from "@/pages/software/detail";

export const Route = createFileRoute("/_authenticated/software/titles/$id")({
  staticData: { breadcrumb: SoftwareBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(softwareTitleQueryOptions(parseRouteID(params.id)));
  },
  component: SoftwareDetailPage,
});

function SoftwareBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/software/titles/$id" });
  const { data } = useQuery(softwareTitleQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
