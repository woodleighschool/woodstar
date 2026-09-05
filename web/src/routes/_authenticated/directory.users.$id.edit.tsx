import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { UserEditPage } from "@features/directory/users/edit";

export const Route = createFileRoute("/_authenticated/directory/users/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "users", access: "edit" }, () => {
      throw redirect({
        to: "/directory/users/$id",
        params: { id: params.id },
      });
    }),
  component: UserEditPage,
});
