import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { LabelEditPage } from "@features/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "labels", access: "edit" }, () => {
      throw redirect({
        to: "/labels/$id",
        params: { id: params.id },
      });
    }),
  component: LabelEditPage,
});
