import { createFileRoute } from "@tanstack/react-router";

import { AccountPage } from "@/pages/account";

export const Route = createFileRoute("/_authenticated/account")({
  staticData: { breadcrumb: "Account" },
  component: AccountPage,
});
