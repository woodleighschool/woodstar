import { createFileRoute } from "@tanstack/react-router";

import { SantaConfigurationResourcePage } from "@/pages/santa/configurations/form";

export const Route = createFileRoute("/_authenticated/santa/configurations/$configurationId")({
  component: () => <SantaConfigurationResourcePage mode="edit" />,
});
