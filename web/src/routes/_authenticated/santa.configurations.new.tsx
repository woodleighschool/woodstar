import { createFileRoute } from "@tanstack/react-router";

import { SantaConfigurationResourcePage } from "@/pages/santa/configurations/form";

export const Route = createFileRoute("/_authenticated/santa/configurations/new")({
  component: () => <SantaConfigurationResourcePage mode="create" />,
});
