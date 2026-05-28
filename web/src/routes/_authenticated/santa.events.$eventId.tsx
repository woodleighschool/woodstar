import { createFileRoute } from "@tanstack/react-router";

import { SantaEventDetailPage } from "@/pages/santa/events/detail";

export const Route = createFileRoute("/_authenticated/santa/events/$eventId")({
  component: SantaEventDetailPage,
});
