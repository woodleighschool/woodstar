import { createRouter } from "@tanstack/react-router";

import { NotFoundPage } from "./not-found";
import { queryClient } from "./query-client";
import { RouteErrorPage } from "./route-error";
import { routeTree } from "./routeTree.gen";

export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: "intent",
  // A revisited route waits for its loader as a first visit does, so a record
  // that has since gone is not found instead of rendered from the cache.
  defaultStaleReloadMode: "blocking",
  defaultErrorComponent: RouteErrorPage,
  defaultNotFoundComponent: NotFoundPage,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
