import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/munki/software")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.software", access: "view" }),
  staticData: { breadcrumb: "Software" },
});
