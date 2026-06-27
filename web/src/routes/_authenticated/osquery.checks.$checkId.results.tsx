import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";
import { z } from "zod";

const searchSchema = z.object({
  response: z.enum(["pass", "fail"]).optional(),
});

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId/results")({
  validateSearch: (search) => searchSchema.parse(search),
  component: lazyRouteComponent(() => import("@/pages/osquery/checks/results"), "CheckResultsPage"),
});
