import { createFileRoute } from "@tanstack/react-router";

import { softwareTitleQueryOptions } from "@/lib/queries/software";
import { parseRouteID } from "@/lib/route-params";
import { SoftwareDetailPage } from "@/pages/software/detail";

export const Route = createFileRoute("/_authenticated/software/titles/$softwareId")({
  loader: async ({ context, params }) => {
    const software = await context.queryClient.ensureQueryData(
      softwareTitleQueryOptions(parseRouteID(params.softwareId)),
    );
    return { breadcrumb: software.name };
  },
  component: SoftwareDetailPage,
});
