import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { reportQueryOptions } from "@/lib/queries/osquery";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/reports/$reportId")({
  staticData: { breadcrumb: ReportBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(reportQueryOptions(parseRouteID(params.reportId)));
  },
});

function ReportBreadcrumb(): string {
  const { reportId } = useParams({ from: "/_authenticated/osquery/reports/$reportId" });
  const { data } = useQuery(reportQueryOptions(parseRouteID(reportId)));
  return data?.name ?? reportId;
}
