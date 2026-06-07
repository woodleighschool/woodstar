import { createFileRoute } from "@tanstack/react-router";

import { MunkiSoftwareDetailPage } from "@/pages/munki/software/detail";

export const Route = createFileRoute("/_authenticated/munki/software/$softwareId")({
  component: MunkiSoftwareDetailPage,
});
