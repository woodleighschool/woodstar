import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { MunkiPackageEditPage } from "@features/munki/packages/edit";
import { PACKAGE_FORM_TAB_SEARCH_VALUES } from "@features/munki/packages/tab-values";

const searchSchema = z.object({
  tab: z.enum(PACKAGE_FORM_TAB_SEARCH_VALUES).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/munki/packages/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/munki/packages/$id",
        params: { id: params.id },
      });
    }),
  component: MunkiPackageEditPage,
});
