import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { reportQueryOptions } from "@/lib/queries/osquery";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id")({
  staticData: { breadcrumb: ReportBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(reportQueryOptions(parseRouteID(params.id)));
  },
});

function ReportBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/osquery/reports/$id" });
  const { data } = useQuery(reportQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
