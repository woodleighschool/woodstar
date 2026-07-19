import { createFileRoute } from "@tanstack/react-router";

import { hostQueryOptions } from "@/lib/queries/hosts";
import { parseRouteID } from "@/lib/route-params";
import { HostDetailPage } from "@/pages/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$hostId")({
  loader: async ({ context, params }) => {
    const host = await context.queryClient.ensureQueryData(
      hostQueryOptions(parseRouteID(params.hostId)),
    );
    return { breadcrumb: host.display_name };
  },
  component: HostDetailPage,
});
