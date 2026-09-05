import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { HostPoliciesPage } from "@features/hosts/detail";
import { POLICY_RESULT_STATUS_VALUES } from "@features/osquery/policies/model";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const SEARCH_DEFAULTS = {
  ...TABLE_SEARCH_DEFAULTS,
  sort: "policy_name.asc",
} as const;

const searchSchema = createTableSearchSchema(["policy_name", "status", "updated_at"], {
  defaultSort: SEARCH_DEFAULTS.sort,
}).extend({
  status: z.array(z.enum(POLICY_RESULT_STATUS_VALUES)).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/hosts/$id/policies")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(SEARCH_DEFAULTS)] },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "osquery.policies", access: "view" }),
  component: HostPoliciesPage,
});
