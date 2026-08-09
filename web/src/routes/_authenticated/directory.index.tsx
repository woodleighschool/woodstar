import { createFileRoute } from "@tanstack/react-router";

import { DirectoryOverviewPage } from "@features/directory/sync/page";

export const Route = createFileRoute("/_authenticated/directory/")({
  component: DirectoryOverviewPage,
});
