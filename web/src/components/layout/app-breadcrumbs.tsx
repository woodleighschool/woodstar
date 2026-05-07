import { Link, useMatches } from "@tanstack/react-router";
import { Fragment, type ReactNode } from "react";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Skeleton } from "@/components/ui/skeleton";
import { useHost } from "@/hooks/use-hosts";
import { useSoftwareTitle } from "@/hooks/use-software";

interface Crumb {
  key: string;
  label: ReactNode;
  to?: string;
  params?: Record<string, string>;
}

export function AppBreadcrumbs() {
  const matches = useMatches();
  const leaf = matches[matches.length - 1] as { routeId: string; params: Record<string, string> } | undefined;
  const crumbs = leaf ? crumbsForLeaf(leaf.routeId, leaf.params) : [];

  if (crumbs.length === 0) return null;

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {crumbs.map((crumb, i) => {
          const isLast = i === crumbs.length - 1;
          return (
            <Fragment key={crumb.key}>
              <BreadcrumbItem>
                {isLast || !crumb.to ? (
                  <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink asChild>
                    <Link to={crumb.to} params={crumb.params}>
                      {crumb.label}
                    </Link>
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

/**
 * Returns the full breadcrumb chain for the deepest matched route.
 * Each top-level page returns one crumb; detail pages return their parent + leaf.
 */
function crumbsForLeaf(routeId: string, params: Record<string, string>): Crumb[] {
  switch (routeId) {
    case "/_authenticated/hosts/":
      return [{ key: "hosts", label: "Hosts" }];
    case "/_authenticated/hosts/$hostId":
      return [
        { key: "hosts", label: "Hosts", to: "/hosts" },
        { key: `host-${params.hostId}`, label: <HostCrumb id={params.hostId} /> },
      ];
    case "/_authenticated/software/":
      return [{ key: "software", label: "Software" }];
    case "/_authenticated/software/titles/$softwareId":
      return [
        { key: "software", label: "Software", to: "/software" },
        { key: `software-${params.softwareId}`, label: <SoftwareCrumb id={params.softwareId} /> },
      ];
    case "/_authenticated/labels":
      return [{ key: "labels", label: "Labels" }];
    case "/_authenticated/users":
      return [{ key: "users", label: "Users" }];
    case "/_authenticated/settings":
      return [{ key: "settings", label: "Settings" }];
    default:
      return [];
  }
}

function HostCrumb({ id }: { id: string }) {
  const { data, isLoading } = useHost(id);
  if (isLoading || !data) return <Skeleton className="inline-block h-4 w-24 align-middle" />;
  return <span title={data.hardware_uuid}>{data.display_name || id}</span>;
}

function SoftwareCrumb({ id }: { id: string }) {
  const { data, isLoading } = useSoftwareTitle(id);
  if (isLoading || !data) return <Skeleton className="inline-block h-4 w-24 align-middle" />;
  return <span>{data.software_title.display_name || data.software_title.name || id}</span>;
}
