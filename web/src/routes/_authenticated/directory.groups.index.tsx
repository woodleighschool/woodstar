import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { GroupListPage } from "@features/directory/groups/list";
import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@lib/list-search";

const searchSchema = createListSearchSchema([
  "display_name",
  "mail_nickname",
  "member_count",
  "source",
]);

export const Route = createFileRoute("/_authenticated/directory/groups/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: GroupListPage,
});
