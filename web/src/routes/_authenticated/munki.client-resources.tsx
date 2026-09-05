import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { MunkiClientResourcesEditPage } from "@features/munki/client-resources/edit";

export const Route = createFileRoute("/_authenticated/munki/client-resources")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.client-resources", access: "view" }),
  staticData: { breadcrumb: "Client Resources" },
  component: MunkiClientResourcesEditPage,
});
