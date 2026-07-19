import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/labels")({
  staticData: { breadcrumb: "Labels" },
});
