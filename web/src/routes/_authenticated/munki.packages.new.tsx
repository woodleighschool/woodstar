import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";
import { z } from "zod";

const searchSchema = z.object({
  software_id: z.coerce.number().int().positive().optional(),
});

export const Route = createFileRoute("/_authenticated/munki/packages/new")({
  validateSearch: (search) => searchSchema.parse(search),
  component: lazyRouteComponent(
    () => import("@/pages/munki/packages/create"),
    "MunkiPackageCreatePage",
  ),
});
