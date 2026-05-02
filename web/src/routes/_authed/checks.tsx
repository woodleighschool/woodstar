import { createFileRoute } from "@tanstack/react-router";

import { ChecksPage } from "@/pages/checks";

export const Route = createFileRoute("/_authed/checks")({
  component: ChecksPage,
});
