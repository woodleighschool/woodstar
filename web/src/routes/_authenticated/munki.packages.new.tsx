import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { MunkiPackageCreatePage } from "@features/munki/packages/create";

const searchSchema = z.object({
  software_id: z.coerce.number().int().positive().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/packages/new")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  beforeLoad: ({ context }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({ to: "/munki/packages" });
    }),
  component: MunkiPackageCreatePage,
});
