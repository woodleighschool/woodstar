import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/directory/users")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "users", access: "view" }),
  staticData: { breadcrumb: "Users" },
});
