import { createFileRoute } from "@tanstack/react-router";

import { AppLayout } from "@/layouts/app-layout";

export const Route = createFileRoute("/_authed")({
  component: AppLayout,
});
