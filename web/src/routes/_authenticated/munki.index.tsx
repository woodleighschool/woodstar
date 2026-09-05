import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { MunkiOverviewPage } from "@features/munki/overview";

export const Route = createFileRoute("/_authenticated/munki/")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.software", access: "view" }),
  component: MunkiOverviewPage,
});
