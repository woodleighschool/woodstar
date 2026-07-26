import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { ConfigurationEditPage } from "@features/santa/configurations/edit";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/santa/configurations/$id",
        params: { id: params.id },
      });
    }),
  component: ConfigurationEditPage,
});
