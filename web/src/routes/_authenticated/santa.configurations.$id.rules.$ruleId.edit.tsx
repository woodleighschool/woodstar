import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { RuleEditPage } from "@features/santa/rules/edit";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/rules/$ruleId/edit")(
  {
    staticData: { breadcrumb: "Edit" },
    validateSearch: searchSchema,
    beforeLoad: ({ context, params }) =>
      requireAdmin(context.currentUser, () => {
        throw redirect({
          to: "/santa/configurations/$id/rules/$ruleId",
          params: { id: params.id, ruleId: params.ruleId },
        });
      }),
    component: RuleEditPage,
  },
);
