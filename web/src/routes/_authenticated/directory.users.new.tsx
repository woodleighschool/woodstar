import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { UserCreatePage } from "@features/directory/users/create";

export const Route = createFileRoute("/_authenticated/directory/users/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "users", access: "edit" }, () => {
      throw redirect({ to: "/directory/users" });
    }),
  component: UserCreatePage,
});
