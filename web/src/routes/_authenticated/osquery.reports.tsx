import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/osquery/reports")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.reports", access: "view" }),
  staticData: { breadcrumb: "Reports" },
});
