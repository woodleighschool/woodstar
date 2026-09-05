import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { HostActivityPage } from "@features/activity/host-page";
import { requirePermission } from "@features/authn/guards";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@lib/pagination";

const searchDefaults = { page: 1, per_page: DEFAULT_PAGE_SIZE };
const searchSchema = z.object({
  page: z.coerce.number().int().min(1).default(1).catch(1),
  per_page: z.coerce
    .number()
    .int()
    .min(1)
    .max(MAX_PAGE_SIZE)
    .default(DEFAULT_PAGE_SIZE)
    .catch(DEFAULT_PAGE_SIZE),
});

export const Route = createFileRoute("/_authenticated/hosts/$id/activity")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(searchDefaults)] },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "activity", access: "view" }),
  component: HostActivityPage,
});
