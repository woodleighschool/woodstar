import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { GroupsPage } from "@/pages/groups/list";

const searchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/directory/groups/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: GroupsPage,
});
