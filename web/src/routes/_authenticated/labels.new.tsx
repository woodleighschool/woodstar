import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { LabelCreatePage } from "@features/labels/create";

export const Route = createFileRoute("/_authenticated/labels/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "labels", access: "edit" }, () => {
      throw redirect({ to: "/labels" });
    }),
  component: LabelCreatePage,
});
