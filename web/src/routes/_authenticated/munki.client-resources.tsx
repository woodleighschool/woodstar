import { createFileRoute } from "@tanstack/react-router";

import { MunkiClientResourcesEditPage } from "@/pages/munki/client-resources/edit";

export const Route = createFileRoute("/_authenticated/munki/client-resources")({
  staticData: { breadcrumb: "Client Resources" },
  component: MunkiClientResourcesEditPage,
});
