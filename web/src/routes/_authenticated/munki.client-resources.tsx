import { createFileRoute } from "@tanstack/react-router";

import { MunkiClientResourcesEditPage } from "@features/munki/client-resources/edit";

export const Route = createFileRoute("/_authenticated/munki/client-resources")({
  staticData: { breadcrumb: "Client Resources" },
  component: MunkiClientResourcesEditPage,
});
