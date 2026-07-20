import type { ComponentType } from "react";

export type BreadcrumbLabel = string | ComponentType;

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption {
    breadcrumb?: BreadcrumbLabel;
  }
}
