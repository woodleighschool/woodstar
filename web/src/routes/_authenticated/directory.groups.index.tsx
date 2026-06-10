import { createFileRoute } from "@tanstack/react-router";

import { tableSearchSchema } from "@/lib/pagination";
import { GroupListPage } from "@/pages/groups/list";

export const Route = createFileRoute("/_authenticated/directory/groups/")({
  validateSearch: (search) => tableSearchSchema.parse(search),
  component: GroupListPage,
});
