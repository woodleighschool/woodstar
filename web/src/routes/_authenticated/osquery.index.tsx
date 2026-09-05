import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { OsqueryOverviewPage } from "@features/osquery/overview";

export const Route = createFileRoute("/_authenticated/osquery/")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.overview", access: "view" }),
  component: OsqueryOverviewPage,
});
