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
import { parseRouteID } from "@lib/route-params";
import { cn } from "@lib/utils";

// Passes a resource route's name to its children once the resource has loaded.
export type ResourceName = ComponentType<{
  id: string | undefined;
  children: (name: string | undefined) => ReactNode;
}>;

type BreadcrumbLabel = string | ResourceName;

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption {
    breadcrumb?: BreadcrumbLabel;
  }
}

export function resourceName<TData, TError, TQueryKey extends QueryKey>(
  options: (id: number | null) => UseQueryOptions<TData, TError, TData, TQueryKey>,
  name: (resource: TData) => string | undefined,
): ResourceName {
  return function ResourceName({ id, children }) {
    const { data } = useQuery(options(parseRouteID(id)));
    return children(data === undefined ? undefined : name(data));
  };
}

export function useBreadcrumbs() {
  return useMatches({
    select: (matches) =>
      matches.flatMap((match) => {
        const label = match.staticData.breadcrumb;
        if (!label) return [];
        const id = "id" in match.params ? match.params.id : undefined;
        return [{ key: match.id, label, to: match.pathname, id }];
      }),
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
                    <BreadcrumbContent label={crumb.label} id={crumb.id} />
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={crumb.to} />}>
                    <BreadcrumbContent label={crumb.label} id={crumb.id} />
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

function BreadcrumbContent({ label, id }: { label: BreadcrumbLabel; id: string | undefined }) {
  if (typeof label === "string") return label;
  const Name = label;
  return <Name id={id}>{(name) => name ?? id}</Name>;
}
