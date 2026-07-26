import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { ReportLivePage } from "@features/osquery/reports/live";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/live")({
  staticData: { breadcrumb: "Live" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/reports/$id",
        params: { id: params.id },
      });
    }),
  component: ReportLivePage,
});
