import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { DistributionPointEditPage } from "@features/munki/distribution-points/edit";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/munki/distribution-points/$id",
        params: { id: params.id },
      });
    }),
  component: DistributionPointEditPage,
});
