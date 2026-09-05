import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { MunkiSoftwareEditPage } from "@features/munki/software/edit";

const searchSchema = z.object({
  tab: z.enum(["targets", "packages"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/software/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "munki.software", access: "edit" }, () => {
      throw redirect({
        to: "/munki/software/$id",
        params: { id: params.id },
      });
    }),
  component: MunkiSoftwareEditPage,
});
