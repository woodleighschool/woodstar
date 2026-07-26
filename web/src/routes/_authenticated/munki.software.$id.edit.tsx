import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { MunkiSoftwareEditPage } from "@features/munki/software/edit";

export const Route = createFileRoute("/_authenticated/munki/software/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/munki/software/$id",
        params: { id: params.id },
      });
    }),
  component: MunkiSoftwareEditPage,
});
