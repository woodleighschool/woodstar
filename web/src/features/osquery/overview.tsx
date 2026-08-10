import { ClipboardCheck, ExternalLink, FileChartColumn } from "lucide-react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { ResourceOverviewCard } from "@components/resource-overview-card";
import { buttonVariants } from "@components/ui/button";
import { EnrollmentOverviewCard } from "@features/enrollments/card";
import { enrollmentDocsURL } from "@features/enrollments/metadata";
import { useChecks } from "@features/osquery/checks/queries";
import { useReports } from "@features/osquery/reports/queries";

const OVERVIEW_COUNT_PARAMS = { page: 1, per_page: 1 } as const;

export function OsqueryOverviewPage() {
  const reports = useReports(OVERVIEW_COUNT_PARAMS);
  const checks = useChecks(OVERVIEW_COUNT_PARAMS);

  return (
    <PageShell>
      <PageHeader
        title="osquery"
        description="Enroll hosts and manage scheduled reports and checks."
        actions={
          <a
            href={enrollmentDocsURL("orbit")}
            target="_blank"
            rel="noreferrer"
            className={buttonVariants({ variant: "outline", size: "sm" })}
          >
            <ExternalLink data-icon="inline-start" />
            Documentation
          </a>
        }
      />

      <div className="grid min-w-0 gap-4 md:grid-cols-3">
        <Link to="/osquery/reports">
          <ResourceOverviewCard
            title="Reports"
            count={reports.data?.count}
            loading={reports.isLoading}
            error={reports.error}
            icon={FileChartColumn}
          />
        </Link>
        <Link to="/osquery/checks">
          <ResourceOverviewCard
            title="Checks"
            count={checks.data?.count}
            loading={checks.isLoading}
            error={checks.error}
            icon={ClipboardCheck}
          />
        </Link>
        <EnrollmentOverviewCard integration="orbit" />
      </div>
    </PageShell>
  );
}
