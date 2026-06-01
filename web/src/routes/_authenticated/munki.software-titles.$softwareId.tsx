import { createFileRoute } from "@tanstack/react-router";

import { MunkiSoftwareTitleDetailPage } from "@/pages/munki/detail";

export const Route = createFileRoute("/_authenticated/munki/software-titles/$softwareId")({
  component: MunkiSoftwareTitleDetailPage,
});
