import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { ReportEditPage } from "@features/osquery/reports/edit";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/reports/$id",
        params: { id: params.id },
      });
    }),
  component: ReportEditPage,
});
