import { createFileRoute } from "@tanstack/react-router";

import { CheckEditPage } from "@/pages/osquery/checks/edit";

export const Route = createFileRoute("/_authenticated/osquery/checks/$id/")({
  component: CheckEditPage,
});
