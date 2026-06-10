import { createFileRoute } from "@tanstack/react-router";

import { tableSearchSchema } from "@/lib/pagination";
import { ConfigurationListPage } from "@/pages/santa/configurations/list";

export const Route = createFileRoute("/_authenticated/santa/configurations/")({
  validateSearch: (search) => tableSearchSchema.parse(search),
  component: ConfigurationListPage,
});
