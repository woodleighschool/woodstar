import { createFileRoute } from "@tanstack/react-router";

import { OsqueryOverviewPage } from "@features/osquery/overview";

export const Route = createFileRoute("/_authenticated/osquery/")({
  component: OsqueryOverviewPage,
});
