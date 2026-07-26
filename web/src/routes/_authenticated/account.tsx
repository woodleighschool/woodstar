import { createFileRoute } from "@tanstack/react-router";

import { AccountPage } from "@features/account/page";

export const Route = createFileRoute("/_authenticated/account")({
  staticData: { breadcrumb: "Account" },
  component: AccountPage,
});
