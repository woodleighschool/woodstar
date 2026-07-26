import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { CheckEditPage } from "@features/osquery/checks/edit";

export const Route = createFileRoute("/_authenticated/osquery/checks/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/checks/$id",
        params: { id: params.id },
      });
    }),
  component: CheckEditPage,
});
