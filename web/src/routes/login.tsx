import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { redirectAuthenticatedFromLogin } from "@features/authn/guards";
import { LoginPage } from "@features/authn/login";

const searchSchema = z.object({
  sso_error: z.string().optional().catch(undefined),
});

export const Route = createFileRoute("/login")({
  validateSearch: searchSchema,
  beforeLoad: ({ context }) => redirectAuthenticatedFromLogin(context.queryClient),
  component: LoginPage,
});
