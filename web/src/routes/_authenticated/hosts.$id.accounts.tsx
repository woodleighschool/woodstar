import { createFileRoute } from "@tanstack/react-router";

import { HostAccountsPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/accounts")({
  component: HostAccountsPage,
});
