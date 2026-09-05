import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/directory/groups")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "groups", access: "view" }),
  staticData: { breadcrumb: "Groups" },
});
