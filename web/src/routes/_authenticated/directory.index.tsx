import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { DirectoryOverviewPage } from "@features/directory/sync/page";

export const Route = createFileRoute("/_authenticated/directory/")({
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "directory", access: "view" }),
  component: DirectoryOverviewPage,
});
