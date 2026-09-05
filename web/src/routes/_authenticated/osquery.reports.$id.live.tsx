import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermissions } from "@features/authn/guards";
import { ReportLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/live")({
  staticData: { breadcrumb: "Live" },
  beforeLoad: ({ context, params }) =>
    requirePermissions(
      context.queryClient,
      [
        { resource: "osquery.reports", access: "view" },
        { resource: "osquery.live-queries", access: "edit" },
      ],
      () => {
        throw redirect({
          to: "/osquery/reports/$id",
          params: { id: params.id },
        });
      },
    ),
  component: ReportLivePage,
});
