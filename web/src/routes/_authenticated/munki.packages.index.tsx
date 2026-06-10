import { createFileRoute } from "@tanstack/react-router";

import { tableSearchSchema } from "@/lib/pagination";
import { MunkiPackageListPage } from "@/pages/munki/packages/list";

export const Route = createFileRoute("/_authenticated/munki/packages/")({
  validateSearch: (search) => tableSearchSchema.parse(search),
  component: MunkiPackageListPage,
});
