import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";
import { canAll } from "@woodleighschool/authz";

import { firstAccessiblePath } from "@components/layout/nav-config";
import { accountQueryOptions } from "@features/account/query-options";
import { sessionQueryOptions } from "@features/authn/queries";
import type { PermissionRequirement } from "@features/authz/permissions";
import type { SessionBody } from "@lib/api";

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

/** Resource route guard backed by the signed-in account's permissions. */
export async function requirePermission(
  queryClient: QueryClient,
  requirement: PermissionRequirement,
  onForbidden?: () => never,
): Promise<void> {
  return requirePermissions(queryClient, [requirement], onForbidden);
}

/** Resource route guard for operations that cross permission boundaries. */
export async function requirePermissions(
  queryClient: QueryClient,
  requirements: readonly PermissionRequirement[],
  onForbidden?: () => never,
): Promise<void> {
  const account = await queryClient.fetchQuery(accountQueryOptions);
  if (canAll(account.effective_permissions, requirements)) return;
  if (onForbidden) return onForbidden();
  throw redirect({ to: firstAccessiblePath(account) ?? "/account" });
}

/** Root entry point: route to login or the app shell. */
export async function redirectForEntry(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.user) throw redirect({ to: "/login" });
  const account = await queryClient.fetchQuery(accountQueryOptions);
  throw redirect({ to: firstAccessiblePath(account) ?? "/account" });
}

/** Login page guard: send an already-signed-in user to the app. */
export async function redirectAuthenticatedFromLogin(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.user) return;
  const account = await queryClient.fetchQuery(accountQueryOptions);
  throw redirect({ to: firstAccessiblePath(account) ?? "/account" });
}
