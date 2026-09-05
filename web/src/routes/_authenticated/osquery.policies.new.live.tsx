import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { PolicyLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/policies/new/live")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.live-queries", access: "edit" }),
  staticData: { breadcrumb: "Live" },
  component: PolicyLivePage,
});
