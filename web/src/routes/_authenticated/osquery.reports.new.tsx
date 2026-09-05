import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/osquery/reports/new")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.reports", access: "edit" }, () => {
      throw redirect({ to: "/osquery/reports" });
    }),
  component: Outlet,
});
