import { redirect } from "@tanstack/react-router";

import { apiClient, unwrap, type Schemas } from "@/lib/api";

export type Session = Schemas["SessionBody"];
export type SessionUser = NonNullable<Session["user"]>;

async function loadSession(): Promise<Session> {
  return unwrap(apiClient.GET("/api/auth/session"));
}

/**
 * Authenticated route guard. Redirects to /setup if setup is incomplete,
 * /login if no user is signed in. Returns the user otherwise.
 */
export async function requireUser(): Promise<SessionUser> {
  const session = await loadSession();
  if (!session.setup_complete) throw redirect({ to: "/setup" });
  if (!session.user) throw redirect({ to: "/login" });
  return session.user;
}

/** Admin-only route guard. Viewers bounce to /hosts. */
export async function requireAdmin(): Promise<SessionUser> {
  const user = await requireUser();
  if (user.role !== "admin") throw redirect({ to: "/hosts" });
  return user;
}

/** Root entry point: route to setup, login, or app shell. */
export async function redirectForEntry(): Promise<void> {
  const session = await loadSession();
  if (!session.setup_complete) throw redirect({ to: "/setup" });
  if (!session.user) throw redirect({ to: "/login" });
  throw redirect({ to: "/hosts" });
}

/** Login page guard: send setup-incomplete to /setup, already-signed-in to /hosts. */
export async function redirectAuthenticatedFromLogin(): Promise<void> {
  const session = await loadSession();
  if (!session.setup_complete) throw redirect({ to: "/setup" });
  if (session.user) throw redirect({ to: "/hosts" });
}

/** Setup page guard: if setup already complete, send the user where they should be. */
export async function redirectCompletedSetup(): Promise<void> {
  const session = await loadSession();
  if (!session.setup_complete) return;
  if (session.user) throw redirect({ to: "/hosts" });
  throw redirect({ to: "/login" });
}
