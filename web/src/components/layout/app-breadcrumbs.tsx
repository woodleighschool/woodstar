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
import { useMunkiSoftwareTitle } from "@/hooks/munki/software-titles";
import { useCheck } from "@/hooks/use-checks";
import { useHost } from "@/hooks/use-hosts";
import { useLabel } from "@/hooks/use-labels";
import { useReport } from "@/hooks/use-reports";
import { useSantaConfiguration, useSantaRule } from "@/hooks/use-santa";
import { useSoftwareTitle } from "@/hooks/use-software";
import { useUser } from "@/hooks/use-users";
import { cn } from "@/lib/utils";

interface Crumb {
  key: string;
  label: ReactNode;
  to?: string;
  params?: Record<string, string>;
}

export function AppBreadcrumbs({ className }: { className?: string }) {
  const matches = useMatches();
  const leaf = matches[matches.length - 1] as { routeId: string; params: Record<string, string> } | undefined;
  const crumbs = leaf ? crumbsForLeaf(leaf.routeId, leaf.params) : [];

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
  if (sidebarRouteIDs.has(routeId)) return [];

  switch (routeId) {
    // Hosts
    case "/_authenticated/hosts/$hostId":
      return [
        { key: "hosts", label: "Hosts", to: "/hosts" },
        { key: `host-${params.hostId}`, label: <HostCrumb id={params.hostId} /> },
      ];

    // Software
    case "/_authenticated/software/titles/$softwareId":
      return [
        { key: "software", label: "Software", to: "/software" },
        { key: `software-${params.softwareId}`, label: <SoftwareCrumb id={params.softwareId} /> },
      ];

    // Labels
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
    case "/_authenticated/osquery/checks/new":
      return [
        { key: "osquery", label: "Osquery", to: "/osquery/reports" },
        { key: "checks", label: "Checks", to: "/osquery/checks" },
        { key: "checks-new", label: "New" },
      ];
    case "/_authenticated/osquery/checks/$checkId/":
      return [
        { key: "osquery", label: "Osquery", to: "/osquery/reports" },
        { key: "checks", label: "Checks", to: "/osquery/checks" },
        { key: `check-${params.checkId}`, label: <CheckCrumb id={params.checkId} /> },
      ];
    // Osquery reports
    case "/_authenticated/osquery/reports/new":
      return [
        { key: "osquery", label: "Osquery", to: "/osquery/reports" },
        { key: "reports", label: "Reports", to: "/osquery/reports" },
        { key: "reports-new", label: "New" },
      ];
    case "/_authenticated/osquery/reports/$reportId/":
      return [
        { key: "osquery", label: "Osquery", to: "/osquery/reports" },
        { key: "reports", label: "Reports", to: "/osquery/reports" },
        { key: `report-${params.reportId}`, label: <ReportCrumb id={params.reportId} /> },
      ];
    case "/_authenticated/osquery/reports/$reportId/live":
      return [
        { key: "osquery", label: "Osquery", to: "/osquery/reports" },
        { key: "reports", label: "Reports", to: "/osquery/reports" },
        {
          key: `report-${params.reportId}`,
          label: <ReportCrumb id={params.reportId} />,
          to: "/osquery/reports/$reportId",
          params: { reportId: params.reportId },
        },
        { key: `report-${params.reportId}-live`, label: "Live" },
      ];

    // Munki
    case "/_authenticated/munki/software-titles/new":
      return [
        { key: "munki", label: "Munki", to: "/munki/software-titles" },
        { key: "munki-software", label: "Software", to: "/munki/software-titles" },
        { key: "munki-software-new", label: "New" },
      ];
    case "/_authenticated/munki/software-titles/$softwareId":
      return [
        { key: "munki", label: "Munki", to: "/munki/software-titles" },
        { key: "munki-software", label: "Software", to: "/munki/software-titles" },
        { key: `munki-software-${params.softwareId}`, label: <MunkiSoftwareCrumb id={params.softwareId} /> },
      ];
    case "/_authenticated/munki/software-titles/$softwareId_/packages/new":
      return [
        { key: "munki", label: "Munki", to: "/munki/software-titles" },
        { key: "munki-software", label: "Software", to: "/munki/software-titles" },
        {
          key: `munki-software-${params.softwareId}`,
          label: <MunkiSoftwareCrumb id={params.softwareId} />,
          to: "/munki/software-titles/$softwareId",
          params: { softwareId: params.softwareId },
        },
        { key: `munki-software-${params.softwareId}-package-new`, label: "New Package" },
      ];
    case "/_authenticated/munki/software-titles/$softwareId_/packages/$packageId/edit":
      return [
        { key: "munki", label: "Munki", to: "/munki/software-titles" },
        { key: "munki-software", label: "Software", to: "/munki/software-titles" },
        {
          key: `munki-software-${params.softwareId}`,
          label: <MunkiSoftwareCrumb id={params.softwareId} />,
          to: "/munki/software-titles/$softwareId",
          params: { softwareId: params.softwareId },
        },
        { key: `munki-package-${params.packageId}`, label: "Edit Package" },
      ];
    // Santa
    case "/_authenticated/santa/configurations/new":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-configurations", label: "Configurations", to: "/santa/configurations" },
        { key: "santa-configurations-new", label: "New" },
      ];
    case "/_authenticated/santa/configurations/$configurationId":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-configurations", label: "Configurations", to: "/santa/configurations" },
        {
          key: `santa-configuration-${params.configurationId}`,
          label: <SantaConfigurationCrumb id={params.configurationId} />,
        },
      ];
    case "/_authenticated/santa/rules/new":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-rules", label: "Rules", to: "/santa/rules" },
        { key: "santa-rules-new", label: "New" },
      ];
    case "/_authenticated/santa/rules/$ruleId":
      return [
        { key: "santa", label: "Santa", to: "/santa/configurations" },
        { key: "santa-rules", label: "Rules", to: "/santa/rules" },
        { key: `santa-rule-${params.ruleId}`, label: <SantaRuleCrumb id={params.ruleId} /> },
      ];

    // Directory
    case "/_authenticated/directory/users/$userId/edit":
      return [
        { key: "directory-users", label: "Users", to: "/directory/users" },
        { key: `user-${params.userId}`, label: <UserCrumb id={params.userId} /> },
      ];
    default:
      return [];
  }
}

const sidebarRouteIDs = new Set([
  "/_authenticated/account",
  "/_authenticated/osquery/checks/",
  "/_authenticated/enrollments",
  "/_authenticated/enrollments/",
  "/_authenticated/enrollments/orbit",
  "/_authenticated/enrollments/santa",
  "/_authenticated/hosts/",
  "/_authenticated/labels/",
  "/_authenticated/munki/software-titles",
  "/_authenticated/munki/software-titles/",
  "/_authenticated/osquery/reports/",
  "/_authenticated/santa/configurations",
  "/_authenticated/santa/configurations/",
  "/_authenticated/santa/events",
  "/_authenticated/santa/events/",
  "/_authenticated/santa/events/file-access",
  "/_authenticated/santa/events/file-access/",
  "/_authenticated/santa/rules",
  "/_authenticated/santa/rules/",
  "/_authenticated/software/",
  "/_authenticated/directory/groups",
  "/_authenticated/directory/groups/",
  "/_authenticated/directory/users",
  "/_authenticated/directory/users/",
]);

function HostCrumb({ id }: { id: string }) {
  const { data, isLoading } = useHost(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span title={data.hardware.uuid}>{data.display_name}</span>;
}

function SoftwareCrumb({ id }: { id: string }) {
  const { data, isLoading } = useSoftwareTitle(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.display_name || data.name || id}</span>;
}

function MunkiSoftwareCrumb({ id }: { id: string }) {
  const { data, isLoading } = useMunkiSoftwareTitle(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function CheckCrumb({ id }: { id: string }) {
  const { data, isLoading } = useCheck(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function ReportCrumb({ id }: { id: string }) {
  const { data, isLoading } = useReport(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function SantaConfigurationCrumb({ id }: { id: string }) {
  const { data, isLoading } = useSantaConfiguration(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function SantaRuleCrumb({ id }: { id: string }) {
  const { data, isLoading } = useSantaRule(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || data.identifier || id}</span>;
}

function LabelCrumb({ id }: { id: string }) {
  const { data, isLoading } = useLabel(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || id}</span>;
}

function UserCrumb({ id }: { id: string }) {
  const { data, isLoading } = useUser(Number(id));
  if (isLoading || !data) return <CrumbSkeleton />;
  return <span>{data.name || data.email || id}</span>;
}

function CrumbSkeleton() {
  return <Skeleton className="inline-block h-4 w-24 align-middle" />;
}
