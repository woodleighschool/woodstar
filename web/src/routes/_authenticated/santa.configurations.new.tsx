import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { ConfigurationCreatePage } from "@features/santa/configurations/create";

export const Route = createFileRoute("/_authenticated/santa/configurations/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/santa/configurations" });
    }),
  component: ConfigurationCreatePage,
});
