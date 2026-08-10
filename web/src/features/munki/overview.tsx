import { Boxes, FileArchive, PackageSearch, RadioTower } from "lucide-react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { ResourceOverviewCard } from "@components/resource-overview-card";
import { AgentSecretsHeaderAction } from "@features/agent-secrets/header-action";
import { useMunkiClientResources } from "@features/munki/client-resources/queries";
import { useMunkiDistributionPoints } from "@features/munki/distribution-points/queries";
import { useMunkiPackages } from "@features/munki/packages/queries";
import { useMunkiSoftware } from "@features/munki/software/queries";

const OVERVIEW_COUNT_PARAMS = { page: 1, per_page: 1 } as const;

export function MunkiOverviewPage() {
  const software = useMunkiSoftware(OVERVIEW_COUNT_PARAMS);
  const packages = useMunkiPackages(OVERVIEW_COUNT_PARAMS);
  const distributionPoints = useMunkiDistributionPoints(OVERVIEW_COUNT_PARAMS);
  const clientResources = useMunkiClientResources();

  return (
    <PageShell>
      <PageHeader
        title="Munki"
        description="Manage software, packages, distribution, and client resources."
        actions={<AgentSecretsHeaderAction agent="munki" />}
      />

      <div className="grid min-w-0 gap-4 md:grid-cols-3">
        <Link to="/munki/software">
          <ResourceOverviewCard
            title="Software"
            count={software.data?.count}
            loading={software.isLoading}
            error={software.error}
            icon={PackageSearch}
          />
        </Link>
        <Link to="/munki/packages">
          <ResourceOverviewCard
            title="Packages"
            count={packages.data?.count}
            loading={packages.isLoading}
            error={packages.error}
            icon={Boxes}
          />
        </Link>
        <Link to="/munki/distribution-points">
          <ResourceOverviewCard
            title="Distribution Points"
            count={distributionPoints.data?.count}
            loading={distributionPoints.isLoading}
            error={distributionPoints.error}
            icon={RadioTower}
          />
        </Link>
        <Link to="/munki/client-resources">
          <ResourceOverviewCard
            title="Client Resources"
            count={clientResources.data?.count}
            loading={clientResources.isLoading}
            error={clientResources.error}
            icon={FileArchive}
          />
        </Link>
      </div>
    </PageShell>
  );
}
