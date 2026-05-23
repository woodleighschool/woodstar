import { createFileRoute } from "@tanstack/react-router";

import { SantaOverviewPage } from "@/pages/santa/overview";

export const Route = createFileRoute("/_authenticated/santa/")({
  component: SantaOverviewPage,
});
