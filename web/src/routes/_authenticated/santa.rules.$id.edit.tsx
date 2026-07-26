import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireAdmin } from "@features/auth/guards";
import { RuleEditPage } from "@features/santa/rules/edit";

export const Route = createFileRoute("/_authenticated/santa/rules/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/santa/rules/$id",
        params: { id: params.id },
      });
    }),
  component: RuleEditPage,
});
