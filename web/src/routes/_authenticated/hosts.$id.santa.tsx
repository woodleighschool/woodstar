import { createFileRoute } from "@tanstack/react-router";

import { requirePermissions } from "@features/authn/guards";
import { HostSantaPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/santa")({
  beforeLoad: ({ context }) =>
    requirePermissions(context.queryClient, [
      { resource: "santa.configurations", access: "view" },
      { resource: "santa.rules", access: "view" },
    ]),
  component: HostSantaPage,
});
