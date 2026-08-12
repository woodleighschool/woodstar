import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { PolicyLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/policies/$id/live")({
  staticData: { breadcrumb: "Live" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/policies/$id",
        params: { id: params.id },
      });
    }),
  component: PolicyLivePage,
});
