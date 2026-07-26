import { createFileRoute } from "@tanstack/react-router";

import { MunkiSoftwareDetailPage } from "@features/munki/software/detail";

export const Route = createFileRoute("/_authenticated/munki/software/$id/")({
  component: MunkiSoftwareDetailPage,
});
