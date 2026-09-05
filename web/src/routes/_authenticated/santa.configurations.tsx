import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/santa/configurations")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "santa.configurations", access: "view" }),
  staticData: { breadcrumb: "Configurations" },
});
