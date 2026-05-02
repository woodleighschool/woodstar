import { createFileRoute } from "@tanstack/react-router";

import { SoftwarePage } from "@/pages/software";

export const Route = createFileRoute("/_authed/software")({
  component: SoftwarePage,
});
