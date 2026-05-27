import { createFileRoute } from "@tanstack/react-router";

import { ChecksPage } from "@/pages/osquery/checks/list";

export const Route = createFileRoute("/_authenticated/osquery/checks/")({
  component: ChecksPage,
});
