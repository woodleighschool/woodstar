import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import {
  ACTIVITY_ACTION_VALUES,
  ACTIVITY_AREA_VALUES,
  ACTIVITY_SCOPE_VALUES,
} from "@features/activity/metadata";
import { ActivityPage } from "@features/activity/page";
import { requirePermission } from "@features/authn/guards";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@lib/pagination";

const searchDefaults = { page: 1, per_page: DEFAULT_PAGE_SIZE, scope: "user" as const };
const searchSchema = z
  .object({
    q: z.string().trim().min(1).optional().catch(undefined),
    page: z.coerce.number().int().min(1).default(1).catch(1),
    per_page: z.coerce
      .number()
      .int()
      .min(1)
      .max(MAX_PAGE_SIZE)
      .default(DEFAULT_PAGE_SIZE)
      .catch(DEFAULT_PAGE_SIZE),
    scope: z.enum(ACTIVITY_SCOPE_VALUES).default("user").catch("user"),
    area: z.enum(ACTIVITY_AREA_VALUES).optional().catch(undefined),
    action: z.enum(ACTIVITY_ACTION_VALUES).optional().catch(undefined),
    from: z.iso.date().optional().catch(undefined),
    to: z.iso.date().optional().catch(undefined),
  })
  .transform((search) => {
    if (search.to && !search.from) return { ...search, to: undefined };
    if (search.from && search.to && search.from > search.to) {
      return { ...search, from: search.to, to: search.from };
    }
    return search;
  });

export const Route = createFileRoute("/_authenticated/activity")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(searchDefaults)] },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "activity", access: "view" }),
  staticData: { breadcrumb: "Activity" },
  component: ActivityPage,
});
