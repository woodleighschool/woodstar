import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { LabelCreatePage } from "@features/labels/create";

export const Route = createFileRoute("/_authenticated/labels/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/labels" });
    }),
  component: LabelCreatePage,
});
