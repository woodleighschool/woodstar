import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { DIRECTORY_SOURCE_VALUES } from "@/lib/directory";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { USER_ACCESS_ROLE_VALUES } from "@/lib/users";
import { UserListPage } from "@/pages/users/list";

const searchSchema = createListSearchSchema([
  "name",
  "email",
  "role",
  "department",
  "created_at",
  "updated_at",
]).extend({
  role: z.enum(USER_ACCESS_ROLE_VALUES).optional().catch(undefined),
  source: z.enum(DIRECTORY_SOURCE_VALUES).optional().catch(undefined),
  group_id: z.coerce.number().int().positive().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/directory/users/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: UserListPage,
});
