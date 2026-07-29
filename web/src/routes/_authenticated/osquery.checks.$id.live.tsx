import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { CheckLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/checks/$id/live")({
  staticData: { breadcrumb: "Live" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/checks/$id",
        params: { id: params.id },
      });
    }),
  component: CheckLivePage,
});
