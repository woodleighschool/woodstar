import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { MunkiPackageCreatePage } from "@features/munki/packages/create";
import { PACKAGE_FORM_TAB_SEARCH_VALUES } from "@features/munki/packages/tab-values";

const searchSchema = z.object({
  software_id: z.coerce.number().int().positive().optional().catch(undefined),
  tab: z.enum(PACKAGE_FORM_TAB_SEARCH_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/packages/new")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "munki.packages", access: "edit" }, () => {
      throw redirect({ to: "/munki/packages" });
    }),
  component: MunkiPackageCreatePage,
});
