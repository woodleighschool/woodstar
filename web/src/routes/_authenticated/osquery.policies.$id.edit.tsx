import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { PolicyEditPage } from "@features/osquery/policies/edit";

const searchSchema = z.object({
  tab: z.enum(["targets", "remediation"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/policies/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/policies/$id",
        params: { id: params.id },
      });
    }),
  component: PolicyEditPage,
});
