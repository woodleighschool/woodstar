import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { HostsListPage } from "@/pages/hosts/list";

const searchSchema = z.object({
  q: z.string().optional(),
  page: z.coerce.number().int().min(1).optional(),
  per_page: z.coerce.number().int().min(10).max(200).optional(),
  order_key: z.string().optional(),
  order_direction: z.enum(["asc", "desc"]).optional(),
  status: z.string().optional(),
  platform: z.string().optional(),
  label_id: z.string().optional(),
  software_title_id: z.string().optional(),
  software_id: z.string().optional(),
});

export const Route = createFileRoute("/_authed/hosts/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: function HostsIndexRoute() {
    const search = Route.useSearch();
    const navigate = Route.useNavigate();

    return (
      <HostsListPage
        search={search}
        setSearch={(updater) => {
          void navigate({ search: updater, replace: true });
        }}
      />
    );
  },
});
