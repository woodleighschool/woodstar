import { createFileRoute } from "@tanstack/react-router";

import { MunkiOverviewPage } from "@features/munki/overview";

export const Route = createFileRoute("/_authenticated/munki/")({
  component: MunkiOverviewPage,
});
