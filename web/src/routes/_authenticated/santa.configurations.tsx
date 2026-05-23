import { createFileRoute } from "@tanstack/react-router";

import { SantaConfigurationsPage } from "@/pages/santa/configurations";

export const Route = createFileRoute("/_authenticated/santa/configurations")({
  component: SantaConfigurationsPage,
});
