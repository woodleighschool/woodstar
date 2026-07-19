import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/santa/events/file-access")({
  staticData: { breadcrumb: "File Access" },
});
