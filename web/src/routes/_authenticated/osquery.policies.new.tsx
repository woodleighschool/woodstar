import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/osquery/policies/new")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.policies", access: "edit" }, () => {
      throw redirect({ to: "/osquery/policies" });
    }),
  component: Outlet,
});
