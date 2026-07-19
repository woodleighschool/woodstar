import { createFileRoute } from "@tanstack/react-router";

import { reportQueryOptions } from "@/lib/queries/osquery";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/reports/$reportId")({
  loader: async ({ context, params }) => {
    const report = await context.queryClient.ensureQueryData(
      reportQueryOptions(parseRouteID(params.reportId)),
    );
    return { breadcrumb: report.name };
  },
});
