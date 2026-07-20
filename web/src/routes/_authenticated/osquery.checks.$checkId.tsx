import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { checkQueryOptions } from "@/lib/queries/osquery";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId")({
  staticData: { breadcrumb: CheckBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(checkQueryOptions(parseRouteID(params.checkId)));
  },
});

function CheckBreadcrumb(): string {
  const { checkId } = useParams({ from: "/_authenticated/osquery/checks/$checkId" });
  const { data } = useQuery(checkQueryOptions(parseRouteID(checkId)));
  return data?.name ?? checkId;
}
