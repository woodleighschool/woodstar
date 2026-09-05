import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { RuleCreatePage } from "@features/santa/rules/create";
import { RULE_TYPE_VALUES } from "@features/santa/rules/metadata";

const searchSchema = z.object({
  rule_type: z.enum(RULE_TYPE_VALUES).optional().catch(undefined),
  identifier: z.string().optional().catch(undefined),
  name: z.string().optional().catch(undefined),
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/rules/new")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "santa.rules", access: "edit" }, () => {
      throw redirect({ to: "/santa/rules" });
    }),
  component: RuleCreatePage,
});
