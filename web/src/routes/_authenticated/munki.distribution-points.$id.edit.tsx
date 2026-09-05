import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { DistributionPointEditPage } from "@features/munki/distribution-points/edit";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermission(
      context.queryClient,
      { resource: "munki.distribution-points", access: "edit" },
      () => {
        throw redirect({
          to: "/munki/distribution-points/$id",
          params: { id: params.id },
        });
      },
    ),
  component: DistributionPointEditPage,
});
