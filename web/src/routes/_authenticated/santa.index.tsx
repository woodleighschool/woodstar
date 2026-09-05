import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { SantaOverviewPage } from "@features/santa/overview";

export const Route = createFileRoute("/_authenticated/santa/")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "santa.configurations", access: "view" }),
  component: SantaOverviewPage,
});
