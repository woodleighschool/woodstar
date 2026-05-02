import { createFileRoute } from "@tanstack/react-router";

import { MunkiHomePage } from "@/pages/munki";

export const Route = createFileRoute("/_authed/munki")({
  component: MunkiHomePage,
});
