import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/labels")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "labels", access: "view" }),
  staticData: { breadcrumb: "Labels" },
});
