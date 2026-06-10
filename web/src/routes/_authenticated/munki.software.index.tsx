import { createFileRoute } from "@tanstack/react-router";

import { tableSearchSchema } from "@/lib/pagination";
import { MunkiSoftwareListPage } from "@/pages/munki/software/list";

export const Route = createFileRoute("/_authenticated/munki/software/")({
  validateSearch: (search) => tableSearchSchema.parse(search),
  component: MunkiSoftwareListPage,
});
