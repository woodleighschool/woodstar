import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { LabelEditPage } from "@features/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/labels/$id",
        params: { id: params.id },
      });
    }),
  component: LabelEditPage,
});
