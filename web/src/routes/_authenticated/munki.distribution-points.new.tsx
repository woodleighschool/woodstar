import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { DistributionPointCreatePage } from "@features/munki/distribution-points/create";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/munki/distribution-points" });
    }),
  component: DistributionPointCreatePage,
});
