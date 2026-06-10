import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { tableSearchSchema } from "@/lib/pagination";
import { UserListPage } from "@/pages/users/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  role: z.enum(["admin", "viewer", "none"]).optional(),
  source: z.enum(["local", "entra"]).optional(),
  group_id: z.coerce.number().int().positive().optional(),
});

export const Route = createFileRoute("/_authenticated/directory/users/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: UserListPage,
});
