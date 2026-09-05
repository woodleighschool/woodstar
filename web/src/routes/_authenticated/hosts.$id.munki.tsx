import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { HostMunkiPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/munki")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.software", access: "view" }),
  component: HostMunkiPage,
});
