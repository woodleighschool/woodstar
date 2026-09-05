import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/munki/packages")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.packages", access: "view" }),
  staticData: { breadcrumb: "Packages" },
});
