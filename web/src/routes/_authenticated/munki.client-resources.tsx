import { createFileRoute } from "@tanstack/react-router";

import { MunkiClientResourcesPage } from "@/pages/munki/client-resources/builder";

export const Route = createFileRoute("/_authenticated/munki/client-resources")({
  staticData: { breadcrumb: "Client Resources" },
  component: MunkiClientResourcesPage,
});
