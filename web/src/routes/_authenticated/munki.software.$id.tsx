import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { munkiSoftwareQueryOptions } from "@features/munki/software/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/software/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(munkiSoftwareQueryOptions, (software) => software.name) },
  loader: (ctx) => loadResource(ctx, munkiSoftwareQueryOptions(ctx.params.id)),
});
