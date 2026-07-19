import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";

import type { SessionBody } from "@/lib/api";
import { sessionQueryOptions } from "@/lib/session";

export type { SessionBody };
export type SessionUser = NonNullable<SessionBody["user"]>;

async function loadSession(queryClient: QueryClient): Promise<SessionBody> {
  return queryClient.fetchQuery(sessionQueryOptions);
}

/**
 * Authenticated route guard. Redirects to /setup if setup is incomplete,
 * /login if no user is signed in. Returns the user otherwise.
 */
export async function requireUser(queryClient: QueryClient): Promise<SessionUser> {
  const session = await loadSession(queryClient);
  if (!session.setup_complete) throw redirect({ to: "/setup" });
  if (!session.user) throw redirect({ to: "/login" });
  return session.user;
}

/** Root entry point: route to setup, login, or app shell. */
export async function redirectForEntry(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.setup_complete) throw redirect({ to: "/setup" });
  if (!session.user) throw redirect({ to: "/login" });
  throw redirect({ to: "/hosts" });
}

/** Login page guard: send setup-incomplete to /setup, already-signed-in to /hosts. */
export async function redirectAuthenticatedFromLogin(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.setup_complete) throw redirect({ to: "/setup" });
  if (session.user) throw redirect({ to: "/hosts" });
}

/** Setup page guard: if setup already complete, send the user where they should be. */
export async function redirectCompletedSetup(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.setup_complete) return;
  if (session.user) throw redirect({ to: "/hosts" });
  throw redirect({ to: "/login" });
}
