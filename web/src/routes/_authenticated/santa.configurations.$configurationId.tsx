import { createFileRoute } from "@tanstack/react-router";

import { ConfigurationEditPage } from "@/pages/santa/configurations/edit";

export const Route = createFileRoute("/_authenticated/santa/configurations/$configurationId")({
  component: ConfigurationEditPage,
});
