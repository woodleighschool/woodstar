import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { LabelsPage } from "@/pages/labels";

const searchSchema = z.object({
  q: z.string().optional(),
  page: z.coerce.number().int().min(1).optional(),
  per_page: z.coerce.number().int().min(10).max(200).optional(),
  order_key: z.string().optional(),
  order_direction: z.enum(["asc", "desc"]).optional(),
  kind: z.string().optional(),
  membership_type: z.string().optional(),
  platform: z.string().optional(),
});

export const Route = createFileRoute("/_authed/labels")({
  validateSearch: (search) => searchSchema.parse(search),
  component: function LabelsRoute() {
    const search = Route.useSearch();
    const navigate = Route.useNavigate();

    return (
      <LabelsPage
        search={search}
        setSearch={(updater) => {
          void navigate({ search: updater, replace: true });
        }}
      />
    );
  },
});
