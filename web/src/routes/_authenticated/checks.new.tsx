import { createFileRoute } from "@tanstack/react-router";

import { CheckEditPage } from "@/pages/checks/edit";

export const Route = createFileRoute("/_authenticated/checks/new")({
  component: () => <CheckEditPage mode="create" />,
});
