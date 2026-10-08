import { createFileRoute } from "@tanstack/react-router";

import { NotFoundPage } from "../../not-found";

export const Route = createFileRoute("/_authenticated/$")({
  staticData: { breadcrumb: "Page Not Found" },
  component: NotFoundPage,
});
