import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { MunkiSoftwareEditPage } from "@features/munki/software/edit";

const searchSchema = z.object({
  tab: z.enum(["targets", "packages"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/software/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/munki/software/$id",
        params: { id: params.id },
      });
    }),
  component: MunkiSoftwareEditPage,
});
