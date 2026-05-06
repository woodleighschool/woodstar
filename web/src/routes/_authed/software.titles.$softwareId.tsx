import { createFileRoute } from "@tanstack/react-router";

import { SoftwareTitleDetailPage } from "@/pages/software/title-detail";

export const Route = createFileRoute("/_authed/software/titles/$softwareId")({
  component: SoftwareTitleDetailPage,
});
