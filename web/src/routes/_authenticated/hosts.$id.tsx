import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { HostDetailPage } from "@features/hosts/detail";
import { hostQueryOptions } from "@features/hosts/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/hosts/$id")({
  staticData: { breadcrumb: resourceName(hostQueryOptions, (host) => host.display_name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(hostQueryOptions(parseRouteID(params.id)));
  },
  component: HostDetailPage,
});
