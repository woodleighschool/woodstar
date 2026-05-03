import { createFileRoute } from "@tanstack/react-router";

import { redirectForEntry } from "@/lib/auth-router";

export const Route = createFileRoute("/")({
  beforeLoad: redirectForEntry,
});
