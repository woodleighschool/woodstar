import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { GroupListPage } from "@features/directory/groups/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "display_name",
  "mail_nickname",
  "member_count",
  "source",
]);

export const Route = createFileRoute("/_authenticated/directory/groups/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: GroupListPage,
});
