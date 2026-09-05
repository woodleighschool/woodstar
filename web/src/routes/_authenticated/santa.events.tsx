import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/santa/events")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "santa.events", access: "view" }),
  staticData: { breadcrumb: "Events" },
});
