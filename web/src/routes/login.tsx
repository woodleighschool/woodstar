import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { redirectAuthenticatedFromLogin } from "@/lib/auth-router";
import { LoginPage } from "@/pages/login";

const searchSchema = z.object({
  sso_error: z.string().optional().catch(undefined),
});

export const Route = createFileRoute("/login")({
  validateSearch: searchSchema,
  beforeLoad: ({ context }) => redirectAuthenticatedFromLogin(context.queryClient),
  component: LoginPage,
});
