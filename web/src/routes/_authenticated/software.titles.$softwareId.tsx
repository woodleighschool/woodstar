import { createFileRoute } from "@tanstack/react-router";

import { SoftwareDetailPage } from "@/pages/software/detail";

export const Route = createFileRoute("/_authenticated/software/titles/$softwareId")({
  component: SoftwareDetailPage,
});
