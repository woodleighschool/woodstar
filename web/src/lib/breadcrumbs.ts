export function breadcrumbLabel(loaderData: unknown): string | undefined {
  if (
    loaderData &&
    typeof loaderData === "object" &&
    "breadcrumb" in loaderData &&
    typeof loaderData.breadcrumb === "string"
  ) {
    return loaderData.breadcrumb;
  }
  return undefined;
}

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption {
    breadcrumb?: string;
  }
}
