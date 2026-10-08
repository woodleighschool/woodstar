import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { reportQueryOptions } from "@features/osquery/reports/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id")({
  staticData: { breadcrumb: resourceName(reportQueryOptions, (report) => report.name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(reportQueryOptions(parseRouteID(params.id)));
  },
});
