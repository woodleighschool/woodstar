import { FileLock2, ListChecks, ScrollText, ShieldCheck } from "lucide-react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { ResourceOverviewCard } from "@components/resource-overview-card";
import { AgentSecretsHeaderAction } from "@features/agent-secrets/header-action";
import { useSantaConfigurations } from "@features/santa/configurations/queries";
import { useSantaEvents, useSantaFileAccessEvents } from "@features/santa/events/queries";
import { useSantaRules } from "@features/santa/rules/queries";

const OVERVIEW_COUNT_PARAMS = { page: 1, per_page: 1 } as const;

export function SantaOverviewPage() {
  const configurations = useSantaConfigurations(OVERVIEW_COUNT_PARAMS);
  const rules = useSantaRules(OVERVIEW_COUNT_PARAMS);
  const events = useSantaEvents(OVERVIEW_COUNT_PARAMS);
  const fileAccessEvents = useSantaFileAccessEvents(OVERVIEW_COUNT_PARAMS);

  return (
    <PageShell>
      <PageHeader
        title="Santa"
        description="Manage client policy, rules, and reported events."
        actions={<AgentSecretsHeaderAction agent="santa" />}
      />

      <div className="grid min-w-0 gap-4 md:grid-cols-3">
        <Link to="/santa/configurations">
          <ResourceOverviewCard
            title="Configurations"
            count={configurations.data?.count}
            loading={configurations.isLoading}
            error={configurations.error}
            icon={ShieldCheck}
          />
        </Link>
        <Link to="/santa/rules">
          <ResourceOverviewCard
            title="Rules"
            count={rules.data?.count}
            loading={rules.isLoading}
            error={rules.error}
            icon={ListChecks}
          />
        </Link>
        <Link to="/santa/events">
          <ResourceOverviewCard
            title="Execution Events"
            count={events.data?.count}
            loading={events.isLoading}
            error={events.error}
            icon={ScrollText}
          />
        </Link>
        <Link to="/santa/events/file-access">
          <ResourceOverviewCard
            title="File Access Events"
            count={fileAccessEvents.data?.count}
            loading={fileAccessEvents.isLoading}
            error={fileAccessEvents.error}
            icon={FileLock2}
          />
        </Link>
      </div>
    </PageShell>
  );
}
