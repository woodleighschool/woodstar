import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermissions } from "@features/authn/guards";
import { PolicyLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/policies/$id/live")({
  staticData: { breadcrumb: "Live" },
  beforeLoad: ({ context, params }) =>
    requirePermissions(
      context.queryClient,
      [
        { resource: "osquery.policies", access: "view" },
        { resource: "osquery.live-queries", access: "edit" },
      ],
      () => {
        throw redirect({
          to: "/osquery/policies/$id",
          params: { id: params.id },
        });
      },
    ),
  component: PolicyLivePage,
});
