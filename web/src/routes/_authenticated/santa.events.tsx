import { createFileRoute } from "@tanstack/react-router";

import { SantaEventsPage } from "@/pages/santa/events";

export const Route = createFileRoute("/_authenticated/santa/events")({
  component: SantaEventsPage,
});
