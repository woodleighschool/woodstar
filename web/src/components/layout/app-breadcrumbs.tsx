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
import { useMunkiDistributionPoint } from "@/hooks/use-munki-distribution-points";
import { useMunkiPackage } from "@/hooks/use-munki-packages";
import { useMunkiSoftwareDetail } from "@/hooks/use-munki-software";
import { useReport } from "@/hooks/use-reports";
import { useSantaConfiguration } from "@/hooks/use-santa-configurations";
import { useSantaRule } from "@/hooks/use-santa-rules";
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
  const leaf = matches[matches.length - 1] as
    | { routeId: string; params: Record<string, string> }
    | undefined;
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
    case "/_authenticated/munki/software/new":
      return [
        { key: "munki", label: "Munki", to: "/munki/software" },
        { key: "munki-software", label: "Software", to: "/munki/software" },
        { key: "munki-software-new", label: "New" },
      ];
    case "/_authenticated/munki/software/$softwareId":
      return [
        { key: "munki", label: "Munki", to: "/munki/software" },
        { key: "munki-software", label: "Software", to: "/munki/software" },
        {
          key: `munki-software-${params.softwareId}`,
          label: <MunkiSoftwareCrumb id={params.softwareId} />,
        },
      ];
    case "/_authenticated/munki/packages/new":
      return [
        { key: "munki", label: "Munki", to: "/munki/packages" },
        { key: "munki-packages", label: "Packages", to: "/munki/packages" },
        { key: "munki-package-new", label: "New" },
      ];
    case "/_authenticated/munki/packages/$packageId/edit":
      return [
        { key: "munki", label: "Munki", to: "/munki/packages" },
        { key: "munki-packages", label: "Packages", to: "/munki/packages" },
        {
          key: `munki-package-${params.packageId}`,
          label: <MunkiPackageCrumb id={params.packageId} />,
        },
      ];
    case "/_authenticated/munki/distribution-points/new":
      return [
        { key: "munki", label: "Munki", to: "/munki/distribution-points" },
        {
          key: "munki-distribution-points",
          label: "Distribution Points",
          to: "/munki/distribution-points",
        },
        { key: "munki-distribution-point-new", label: "New" },
      ];
    case "/_authenticated/munki/distribution-points/$distributionPointId":
      return [
        { key: "munki", label: "Munki", to: "/munki/distribution-points" },
        {
          key: "munki-distribution-points",
          label: "Distribution Points",
          to: "/munki/distribution-points",
        },
        {
          key: `munki-distribution-point-${params.distributionPointId}`,
          label: <MunkiDistributionPointCrumb id={params.distributionPointId} />,
        },
      ];
    case "/_authenticated/munki/distribution-points/$distributionPointId/edit":
      return [
        { key: "munki", label: "Munki", to: "/munki/distribution-points" },
        {
          key: "munki-distribution-points",
          label: "Distribution Points",
          to: "/munki/distribution-points",
        },
        {
          key: `munki-distribution-point-${params.distributionPointId}`,
          label: <MunkiDistributionPointCrumb id={params.distributionPointId} />,
          to: "/munki/distribution-points/$distributionPointId",
          params: { distributionPointId: params.distributionPointId },
        },
        { key: `munki-distribution-point-${params.distributionPointId}-edit`, label: "Edit" },
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

function resourceCrumb<T>(
  useDetail: (id: number) => { data?: T; isError?: boolean; isLoading: boolean },
  label: (data: T, id: string) => ReactNode,
): (props: { id: string }) => ReactNode {
  return function ResourceCrumb({ id }: { id: string }) {
    const { data, isError, isLoading } = useDetail(Number(id));
    if (data) return <span>{label(data, id)}</span>;
    if (isLoading) return <CrumbSkeleton />;
    if (isError) return <span>{id}</span>;
    return <CrumbSkeleton />;
  };
}

function HostCrumb({ id }: { id: string }) {
  const { data, isError, isLoading } = useHost(Number(id));
  if (!data) {
    if (isLoading) return <CrumbSkeleton />;
    if (isError) return <span>{id}</span>;
    return <CrumbSkeleton />;
  }
  return <span title={data.hardware.uuid}>{data.display_name}</span>;
}

const SoftwareCrumb = resourceCrumb(useSoftwareTitle, (d, id) => d.display_name || d.name || id);
const MunkiSoftwareCrumb = resourceCrumb(useMunkiSoftwareDetail, (d, id) => d.name || id);
const MunkiDistributionPointCrumb = resourceCrumb(
  useMunkiDistributionPoint,
  (d, id) => d.name || id,
);
const CheckCrumb = resourceCrumb(useCheck, (d, id) => d.name || id);
const ReportCrumb = resourceCrumb(useReport, (d, id) => d.name || id);
const SantaConfigurationCrumb = resourceCrumb(useSantaConfiguration, (d, id) => d.name || id);
const SantaRuleCrumb = resourceCrumb(useSantaRule, (d, id) => d.name || d.identifier || id);
const LabelCrumb = resourceCrumb(useLabel, (d, id) => d.name || id);
const UserCrumb = resourceCrumb(useUser, (d, id) => d.name || d.email || id);

function MunkiPackageCrumb({ id }: { id: string }) {
  const { data, isError, isLoading } = useMunkiPackage(Number(id));
  if (!data) {
    if (isLoading) return <CrumbSkeleton />;
    if (isError) return <span>{id}</span>;
    return <CrumbSkeleton />;
  }
  return <span>{`${data.software_name} ${data.version}`}</span>;
}

function CrumbSkeleton() {
  return <Skeleton className="inline-block h-4 w-24 align-middle" />;
}
