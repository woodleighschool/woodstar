import { createRouter } from "@tanstack/react-router";

import { NotFoundPage } from "./not-found";
import { queryClient } from "./query-client";
import { RouteErrorPage } from "./route-error";
import { routeTree } from "./routeTree.gen";

export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: "intent",
  defaultErrorComponent: RouteErrorPage,
  defaultNotFoundComponent: NotFoundPage,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
