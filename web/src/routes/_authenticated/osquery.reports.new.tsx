import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { ReportCreatePage } from "@features/osquery/reports/create";

export const Route = createFileRoute("/_authenticated/osquery/reports/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/osquery/reports" });
    }),
  component: ReportCreatePage,
});
