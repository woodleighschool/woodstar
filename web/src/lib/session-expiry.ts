import type { QueryClient } from "@tanstack/react-query";

let redirecting = false;

export function expireSession(
  queryClient: QueryClient,
  path: string,
  navigateToLogin: () => Promise<void>,
): void {
  if (path === "/login" || redirecting) return;

  redirecting = true;
  queryClient.clear();
  void navigateToLogin().finally(() => {
    redirecting = false;
  });
}
