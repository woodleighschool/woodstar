import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { MunkiSoftwareCreatePage } from "@features/munki/software/create";

export const Route = createFileRoute("/_authenticated/munki/software/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/munki/software" });
    }),
  component: MunkiSoftwareCreatePage,
});
