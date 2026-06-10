import { createFileRoute } from "@tanstack/react-router";

import { MunkiSoftwareEditPage } from "@/pages/munki/software/edit";

export const Route = createFileRoute("/_authenticated/munki/software/$softwareId")({
  component: MunkiSoftwareEditPage,
});
