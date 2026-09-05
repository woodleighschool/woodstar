import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { MunkiSoftwareCreatePage } from "@features/munki/software/create";

export const Route = createFileRoute("/_authenticated/munki/software/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.software", access: "edit" }, () => {
      throw redirect({ to: "/munki/software" });
    }),
  component: MunkiSoftwareCreatePage,
});
