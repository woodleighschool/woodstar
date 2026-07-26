import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { MunkiPackageEditPage } from "@features/munki/packages/edit";

export const Route = createFileRoute("/_authenticated/munki/packages/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/munki/packages/$id",
        params: { id: params.id },
      });
    }),
  component: MunkiPackageEditPage,
});
