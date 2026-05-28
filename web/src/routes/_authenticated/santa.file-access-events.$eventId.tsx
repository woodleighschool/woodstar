import { createFileRoute } from "@tanstack/react-router";

import { SantaFileAccessEventDetailPage } from "@/pages/santa/file-access-events/detail";

export const Route = createFileRoute("/_authenticated/santa/file-access-events/$eventId")({
  component: SantaFileAccessEventDetailPage,
});
