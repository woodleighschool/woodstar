import { createFileRoute } from "@tanstack/react-router";

import { SantaHomePage } from "@/pages/santa";

export const Route = createFileRoute("/_authed/santa")({
  component: SantaHomePage,
});
