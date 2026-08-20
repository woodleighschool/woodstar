import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { RuleCreatePage } from "@features/santa/rules/create";
import { RULE_TYPE_VALUES } from "@features/santa/rules/metadata";

const searchSchema = z.object({
  rule_type: z.enum(RULE_TYPE_VALUES).optional().catch(undefined),
  identifier: z.string().optional().catch(undefined),
  name: z.string().optional().catch(undefined),
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/rules/new")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/santa/configurations/$id/rules",
        params: { id: params.id },
      });
    }),
  component: RuleCreatePage,
});
