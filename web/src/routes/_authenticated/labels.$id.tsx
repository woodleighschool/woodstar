import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { labelQueryOptions } from "@features/labels/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/labels/$id")({
  staticData: { breadcrumb: resourceName(labelQueryOptions, (label) => label.name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(labelQueryOptions(parseRouteID(params.id)));
  },
});
