import { createFileRoute } from "@tanstack/react-router";

import { SantaConfigurationEditPage } from "@/pages/santa/configurations/edit";

export const Route = createFileRoute("/_authenticated/santa/configurations/new")({
  component: () => <SantaConfigurationEditPage mode="create" />,
});
