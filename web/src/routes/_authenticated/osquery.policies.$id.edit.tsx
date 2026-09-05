import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { PolicyEditPage } from "@features/osquery/policies/edit";

const searchSchema = z.object({
  tab: z.enum(["targets", "remediation"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/policies/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "osquery.policies", access: "edit" }, () => {
      throw redirect({
        to: "/osquery/policies/$id",
        params: { id: params.id },
      });
    }),
  component: PolicyEditPage,
});
