import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { checkQueryOptions } from "@/lib/queries/osquery";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/checks/$id")({
  staticData: { breadcrumb: CheckBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(checkQueryOptions(parseRouteID(params.id)));
  },
});

function CheckBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/osquery/checks/$id" });
  const { data } = useQuery(checkQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
