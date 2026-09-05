import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/software")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "software", access: "view" }),
  staticData: { breadcrumb: "Software" },
});
