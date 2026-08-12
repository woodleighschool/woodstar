import { ClipboardCheck, FileChartColumn } from "lucide-react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { ResourceOverviewCard } from "@components/resource-overview-card";
import { AgentSecretsHeaderAction } from "@features/agent-secrets/header-action";
import { usePolicies } from "@features/osquery/policies/queries";
import { useReports } from "@features/osquery/reports/queries";

const OVERVIEW_COUNT_PARAMS = { page: 1, per_page: 1 } as const;

export function OsqueryOverviewPage() {
  const reports = useReports(OVERVIEW_COUNT_PARAMS);
  const policies = usePolicies(OVERVIEW_COUNT_PARAMS);

  return (
    <PageShell>
      <PageHeader
        title="osquery"
        description="Enroll hosts and manage scheduled reports and policies."
        actions={<AgentSecretsHeaderAction agent="orbit" />}
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
        <Link to="/osquery/policies">
          <ResourceOverviewCard
            title="Policies"
            count={policies.data?.count}
            loading={policies.isLoading}
            error={policies.error}
            icon={ClipboardCheck}
          />
        </Link>
      </div>
    </PageShell>
  );
}
