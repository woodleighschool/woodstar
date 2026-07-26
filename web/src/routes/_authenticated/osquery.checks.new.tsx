import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { CheckCreatePage } from "@features/osquery/checks/create";

export const Route = createFileRoute("/_authenticated/osquery/checks/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/osquery/checks" });
    }),
  component: CheckCreatePage,
});
