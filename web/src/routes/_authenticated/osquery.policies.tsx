import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/osquery/policies")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.policies", access: "view" }),
  staticData: { breadcrumb: "Policies" },
});
