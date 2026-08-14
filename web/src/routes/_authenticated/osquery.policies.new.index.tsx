import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { PolicyCreatePage } from "@features/osquery/policies/create";

const searchSchema = z.object({
  tab: z.enum(["targets", "remediation"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/policies/new/")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  component: PolicyCreatePage,
});
