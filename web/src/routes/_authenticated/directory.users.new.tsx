import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { UserCreatePage } from "@features/directory/users/create";

export const Route = createFileRoute("/_authenticated/directory/users/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/directory/users" });
    }),
  component: UserCreatePage,
});
