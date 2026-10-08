import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { policyQueryOptions } from "@features/osquery/policies/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/policies/$id")({
  staticData: { breadcrumb: resourceName(policyQueryOptions, (policy) => policy.name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(policyQueryOptions(parseRouteID(params.id)));
  },
});
