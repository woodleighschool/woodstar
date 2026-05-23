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
import { useCheck } from "@/hooks/use-checks";
import { useHost } from "@/hooks/use-hosts";
import { useLabel } from "@/hooks/use-labels";
import { useReport } from "@/hooks/use-reports";
import { useSoftwareTitle } from "@/hooks/use-software";
import { useUser } from "@/hooks/use-users";

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

function crumbsForLeaf(routeId: string, params: Record<string, string>): Crumb[] {
  switch (routeId) {
    // Hosts
    case "/_authenticated/hosts/":
      return [{ key: "hosts", label: "Hosts" }];
    case "/_authenticated/hosts/$hostId":
      return [
        { key: "hosts", label: "Hosts", to: "/hosts" },
        { key: `host-${params.hostId}`, label: <HostCrumb id={params.hostId} /> },
      ];
    case "/_authenticated/hosts/$hostId/reports/$reportId":
      return [
        { key: "hosts", label: "Hosts", to: "/hosts" },
        {
          key: `host-${params.hostId}`,
          label: <HostCrumb id={params.hostId} />,
          to: "/hosts/$hostId",
          params: { hostId: params.hostId },
        },
        { key: `report-${params.reportId}`, label: <ReportCrumb id={params.reportId} /> },
      ];

    // Software
    case "/_authenticated/software/":
      return [{ key: "software", label: "Software" }];
    case "/_authenticated/software/titles/$softwareId":
      return [
        { key: "software", label: "Software", to: "/software" },
        { key: `software-${params.softwareId}`, label: <SoftwareCrumb id={params.softwareId} /> },
      ];

    // Labels
    case "/_authenticated/labels/":
      return [{ key: "labels", label: "Labels" }];
    case "/_authenticated/labels/new":
      return [
        { key: "labels", label: "Labels", to: "/labels" },
        { key: "labels-new", label: "New" },
      ];
    case "/_authenticated/labels/$labelId/edit":
      return [
        { key: "labels", label: "Labels", to: "/labels" },
        { key: `label-${params.labelId}`, label: <LabelCrumb id={params.labelId} /> },
      ];

    // Osquery checks
    case "/_authenticated/checks/":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "checks", label: "Checks" },
      ];
    case "/_authenticated/checks/new":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "checks", label: "Checks", to: "/checks" },
        { key: "checks-new", label: "New" },
      ];
    case "/_authenticated/checks/$checkId/":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "checks", label: "Checks", to: "/checks" },
        { key: `check-${params.checkId}`, label: <CheckCrumb id={params.checkId} /> },
      ];
    case "/_authenticated/checks/$checkId/edit":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "checks", label: "Checks", to: "/checks" },
        {
          key: `check-${params.checkId}`,
          label: <CheckCrumb id={params.checkId} />,
          to: "/checks/$checkId",
          params: { checkId: params.checkId },
        },
        { key: `check-${params.checkId}-edit`, label: "Edit" },
      ];

    // Osquery reports
    case "/_authenticated/reports/":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "reports", label: "Reports" },
      ];
    case "/_authenticated/reports/new":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "reports", label: "Reports", to: "/reports" },
        { key: "reports-new", label: "New" },
      ];
    case "/_authenticated/reports/$reportId/":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "reports", label: "Reports", to: "/reports" },
        { key: `report-${params.reportId}`, label: <ReportCrumb id={params.reportId} /> },
      ];
    case "/_authenticated/reports/$reportId/edit":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "reports", label: "Reports", to: "/reports" },
        {
          key: `report-${params.reportId}`,
          label: <ReportCrumb id={params.reportId} />,
          to: "/reports/$reportId",
          params: { reportId: params.reportId },
        },
        { key: `report-${params.reportId}-edit`, label: "Edit" },
      ];
    case "/_authenticated/reports/$reportId/live":
      return [
        { key: "osquery", label: "Osquery", to: "/reports" },
        { key: "reports", label: "Reports", to: "/reports" },
        {
          key: `report-${params.reportId}`,
          label: <ReportCrumb id={params.reportId} />,
          to: "/reports/$reportId",
          params: { reportId: params.reportId },
        },
        { key: `report-${params.reportId}-live`, label: "Live" },
      ];

    // Santa
    case "/_authenticated/santa/configurations":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-configurations", label: "Configurations" },
      ];
    case "/_authenticated/santa/rules":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-rules", label: "Rules" },
      ];
    case "/_authenticated/santa/sync-tokens":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-sync-tokens", label: "Sync tokens" },
      ];
    case "/_authenticated/santa/events":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-events", label: "Events" },
      ];

    // System
    case "/_authenticated/account":
      return [{ key: "account", label: "Account" }];
    case "/_authenticated/users":
      return [{ key: "users", label: "Users" }];
    case "/_authenticated/users/$userId/edit":
      return [
        { key: "users", label: "Users", to: "/users" },
        { key: `user-${params.userId}`, label: <UserCrumb id={params.userId} /> },
      ];
    case "/_authenticated/settings":
      return [{ key: "settings", label: "Settings" }];

    default:
      return [];
  }
}

function HostCrumb({ id }: { id: string }) {
  const { data, isLoading } = useHost(id);
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span title={data.hardware_uuid}>{data.display_name || id}</span>;
}

function SoftwareCrumb({ id }: { id: string }) {
  const { data, isLoading } = useSoftwareTitle(id);
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.display_name || data.name || id}</span>;
}

function CheckCrumb({ id }: { id: string }) {
  const { data, isLoading } = useCheck(id);
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function ReportCrumb({ id }: { id: string }) {
  const { data, isLoading } = useReport(id);
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function LabelCrumb({ id }: { id: string }) {
  const { data, isLoading } = useLabel(id);
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function UserCrumb({ id }: { id: string }) {
  const { data, isLoading } = useUser(id);
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || data.email || id}</span>;
}

function CrumbSkeleton() {
  return <Skeleton className="inline-block h-4 w-24 align-middle" />;
}
