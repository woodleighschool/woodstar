import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { RuleEditPage } from "@features/santa/rules/edit";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/rules/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "santa.rules", access: "edit" }, () => {
      throw redirect({
        to: "/santa/rules/$id",
        params: { id: params.id },
      });
    }),
  component: RuleEditPage,
});
