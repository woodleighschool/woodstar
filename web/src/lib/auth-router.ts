import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";

import type { SessionBody } from "@/lib/api";
import { sessionQueryOptions } from "@/lib/session";

export type { SessionBody };
export type SessionUser = NonNullable<SessionBody["user"]>;

async function loadSession(queryClient: QueryClient): Promise<SessionBody> {
  return queryClient.fetchQuery(sessionQueryOptions);
}

/** Authenticated route guard. Redirects to login if no user is signed in. */
export async function requireUser(queryClient: QueryClient): Promise<SessionUser> {
  const session = await loadSession(queryClient);
  if (!session.user) throw redirect({ to: "/login" });
  return session.user;
}

/** Root entry point: route to login or the app shell. */
export async function redirectForEntry(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.user) throw redirect({ to: "/login" });
  throw redirect({ to: "/hosts" });
}

/** Login page guard: send an already-signed-in user to the app. */
export async function redirectAuthenticatedFromLogin(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (session.user) throw redirect({ to: "/hosts" });
}
