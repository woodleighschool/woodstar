import { createFileRoute } from "@tanstack/react-router";

import { munkiSoftwareQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { MunkiSoftwareEditPage } from "@/pages/munki/software/edit";

export const Route = createFileRoute("/_authenticated/munki/software/$softwareId")({
  loader: async ({ context, params }) => {
    const software = await context.queryClient.ensureQueryData(
      munkiSoftwareQueryOptions(parseRouteID(params.softwareId)),
    );
    return { breadcrumb: software.name };
  },
  component: MunkiSoftwareEditPage,
});
