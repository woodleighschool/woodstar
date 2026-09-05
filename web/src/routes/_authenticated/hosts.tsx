import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/hosts")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "hosts", access: "view" }),
  staticData: { breadcrumb: "Hosts" },
});
