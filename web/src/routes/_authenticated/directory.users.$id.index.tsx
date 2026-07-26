import { createFileRoute } from "@tanstack/react-router";

import { UserDetailPage } from "@features/directory/users/detail";

export const Route = createFileRoute("/_authenticated/directory/users/$id/")({
  component: UserDetailPage,
});
