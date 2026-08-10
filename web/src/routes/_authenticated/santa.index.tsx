import { createFileRoute } from "@tanstack/react-router";

import { SantaOverviewPage } from "@features/santa/overview";

export const Route = createFileRoute("/_authenticated/santa/")({
  component: SantaOverviewPage,
});
