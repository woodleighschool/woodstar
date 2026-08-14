import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { policyQueryOptions } from "@features/osquery/policies/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/policies/$id")({
  staticData: { breadcrumb: PolicyBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(policyQueryOptions(parseRouteID(params.id)));
  },
});

function PolicyBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/osquery/policies/$id" });
  const { data } = useQuery(policyQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
