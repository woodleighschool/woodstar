import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { UsersPage } from "@/pages/users";

const searchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
  role: z.enum(["admin", "viewer", "none"]).optional(),
  source: z.enum(["local", "synced"]).optional(),
  status: z.enum(["active", "inactive"]).optional(),
});

export const Route = createFileRoute("/_authenticated/directory/users/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: UsersPage,
});
