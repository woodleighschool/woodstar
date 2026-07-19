import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/directory")({
  staticData: { breadcrumb: "Directory" },
});
