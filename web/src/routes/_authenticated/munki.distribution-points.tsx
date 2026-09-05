import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/munki/distribution-points")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, {
      resource: "munki.distribution-points",
      access: "view",
    }),
  staticData: { breadcrumb: "Distribution Points" },
});
