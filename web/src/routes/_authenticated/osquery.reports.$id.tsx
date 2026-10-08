import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { reportQueryOptions } from "@features/osquery/reports/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(reportQueryOptions, (report) => report.name) },
  loader: (ctx) => loadResource(ctx, reportQueryOptions(ctx.params.id)),
});
