import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { DistributionPointCreatePage } from "@features/munki/distribution-points/create";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(
      context.queryClient,
      { resource: "munki.distribution-points", access: "edit" },
      () => {
        throw redirect({ to: "/munki/distribution-points" });
      },
    ),
  component: DistributionPointCreatePage,
});
