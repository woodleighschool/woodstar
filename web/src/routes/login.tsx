import { createFileRoute } from "@tanstack/react-router";

import { redirectAuthenticatedFromLogin } from "@/lib/auth-router";
import { LoginPage } from "@/pages/login";

export const Route = createFileRoute("/login")({
  beforeLoad: redirectAuthenticatedFromLogin,
  component: LoginPage,
});
