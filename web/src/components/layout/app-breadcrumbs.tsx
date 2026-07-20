import { Link, useMatches } from "@tanstack/react-router";
import { Fragment } from "react";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import type { BreadcrumbLabel } from "@/lib/breadcrumbs";
import { cn } from "@/lib/utils";

export function AppBreadcrumbs({ className }: { className?: string }) {
  const crumbs = useMatches({
    select: (matches) =>
      matches.flatMap((match) => {
        const label = match.staticData.breadcrumb;
        return label ? [{ key: match.id, label, to: match.pathname }] : [];
      }),
  });

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
                    <BreadcrumbContent label={crumb.label} />
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={crumb.to} />}>
                    <BreadcrumbContent label={crumb.label} />
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

function BreadcrumbContent({ label }: { label: BreadcrumbLabel }) {
  if (typeof label === "string") return label;
  const Label = label;
  return <Label />;
}
