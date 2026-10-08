import { useQuery, type QueryKey, type UseQueryOptions } from "@tanstack/react-query";
import { useMatches } from "@tanstack/react-router";
import { Fragment, type ComponentType, type ReactNode } from "react";

import { Link } from "@components/link";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@components/ui/breadcrumb";
import { cn } from "@lib/utils";

// Passes a resource route's name to its children once the resource has loaded.
export type ResourceName = ComponentType<{
  id: number;
  children: (name: string | undefined) => ReactNode;
}>;

type BreadcrumbLabel = string | ResourceName;

type Crumb = { key: string; to: string } & (
  | { label: string }
  | { label: ResourceName; id: number }
);

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption {
    breadcrumb?: BreadcrumbLabel;
  }
}

export function resourceName<TData, TError, TQueryKey extends QueryKey>(
  options: (id: number) => UseQueryOptions<TData, TError, TData, TQueryKey>,
  name: (resource: TData) => string | undefined,
): ResourceName {
  return function ResourceName({ id, children }) {
    const { data } = useQuery(options(id));
    return children(data === undefined ? undefined : name(data));
  };
}

export function useBreadcrumbs() {
  return useMatches({
    select: (matches) => {
      const crumbs: Crumb[] = [];
      for (const match of matches) {
        const label = match.staticData.breadcrumb;
        const crumb = { key: match.id, to: match.pathname };
        // The not-found page stands in for a missing route and its children.
        if (match.status === "notFound") {
          crumbs.push({ ...crumb, label: "Page Not Found" });
          break;
        }
        if (typeof label === "string") {
          crumbs.push({ ...crumb, label });
        } else if (label && "id" in match.params) {
          // Only routes with an id param declare a resource name.
          crumbs.push({ ...crumb, label, id: match.params.id });
        }
      }
      return crumbs;
    },
  });
}

export function AppBreadcrumbs({ className }: { className?: string }) {
  const crumbs = useBreadcrumbs();

  if (crumbs.length === 0) return null;

  return (
    <Breadcrumb className={cn("min-w-0", className)}>
      <BreadcrumbList>
        {crumbs.map((crumb, i) => {
          const isLast = i === crumbs.length - 1;
          return (
            <Fragment key={crumb.key}>
              <BreadcrumbItem>
                {isLast || !crumb.to ? (
                  <BreadcrumbPage>
                    <BreadcrumbContent crumb={crumb} />
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={crumb.to} />}>
                    <BreadcrumbContent crumb={crumb} />
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
              {!isLast ? <BreadcrumbSeparator /> : null}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}

function BreadcrumbContent({ crumb }: { crumb: Crumb }) {
  if (!("id" in crumb)) return crumb.label;
  const { label: Name, id } = crumb;
  return <Name id={id}>{(name) => name ?? id}</Name>;
}
