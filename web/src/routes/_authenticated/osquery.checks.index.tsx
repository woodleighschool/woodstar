import { createFileRoute } from "@tanstack/react-router";

import { tableSearchSchema } from "@/lib/pagination";
import { CheckListPage } from "@/pages/osquery/checks/list";

export const Route = createFileRoute("/_authenticated/osquery/checks/")({
  validateSearch: (search) => tableSearchSchema.parse(search),
  component: CheckListPage,
});
