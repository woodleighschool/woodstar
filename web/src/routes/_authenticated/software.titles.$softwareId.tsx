import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { softwareTitleQueryOptions } from "@/lib/queries/software";
import { parseRouteID } from "@/lib/route-params";
import { SoftwareDetailPage } from "@/pages/software/detail";

export const Route = createFileRoute("/_authenticated/software/titles/$softwareId")({
  staticData: { breadcrumb: SoftwareBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      softwareTitleQueryOptions(parseRouteID(params.softwareId)),
    );
  },
  component: SoftwareDetailPage,
});

function SoftwareBreadcrumb(): string {
  const { softwareId } = useParams({ from: "/_authenticated/software/titles/$softwareId" });
  const { data } = useQuery(softwareTitleQueryOptions(parseRouteID(softwareId)));
  return data?.name ?? softwareId;
}
