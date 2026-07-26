import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { UserEditPage } from "@features/directory/users/edit";

export const Route = createFileRoute("/_authenticated/directory/users/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/directory/users/$id",
        params: { id: params.id },
      });
    }),
  component: UserEditPage,
});
