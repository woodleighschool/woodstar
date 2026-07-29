import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";

export const Route = createFileRoute("/_authenticated/osquery/checks/new")({
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/osquery/checks" });
    }),
  component: Outlet,
});
