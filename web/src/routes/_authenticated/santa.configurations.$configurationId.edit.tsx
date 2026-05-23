import { createFileRoute } from "@tanstack/react-router";

import { SantaConfigurationEditPage } from "@/pages/santa/configurations";

export const Route = createFileRoute("/_authenticated/santa/configurations/$configurationId/edit")({
  component: () => <SantaConfigurationEditPage mode="edit" />,
});
