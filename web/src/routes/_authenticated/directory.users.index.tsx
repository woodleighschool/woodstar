import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { DIRECTORY_SOURCE_VALUES } from "@features/directory/source";
import { UserListPage } from "@features/directory/users/list";
import { USER_ACCESS_ROLE_VALUES } from "@features/directory/users/metadata";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "name",
  "email",
  "role",
  "department",
  "created_at",
  "updated_at",
]).extend({
  role: z.array(z.enum(USER_ACCESS_ROLE_VALUES)).optional().catch(undefined),
  source: z.enum(DIRECTORY_SOURCE_VALUES).optional().catch(undefined),
  group_id: z.coerce.number().int().positive().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/directory/users/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: UserListPage,
});
