import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { GroupListPage } from "@/pages/groups/list";

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
