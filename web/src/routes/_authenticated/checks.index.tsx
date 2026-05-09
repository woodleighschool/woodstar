import { createFileRoute } from "@tanstack/react-router";

import { ChecksPage } from "@/pages/checks/list";

export const Route = createFileRoute("/_authenticated/checks/")({
  component: ChecksPage,
});
