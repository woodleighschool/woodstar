import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { SoftwareDetailPage } from "@features/software/detail";
import { softwareTitleQueryOptions } from "@features/software/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/software/titles/$id")({
  staticData: { breadcrumb: resourceName(softwareTitleQueryOptions, (title) => title.name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(softwareTitleQueryOptions(parseRouteID(params.id)));
  },
  component: SoftwareDetailPage,
});
