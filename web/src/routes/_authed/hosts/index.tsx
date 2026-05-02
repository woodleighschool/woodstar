import { createFileRoute } from "@tanstack/react-router";

import { HostsListPage } from "@/pages/hosts/list";

export const Route = createFileRoute("/_authed/hosts/")({
  component: HostsListPage,
});
