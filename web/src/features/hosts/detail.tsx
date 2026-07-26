import { Outlet, useParams, useRouterState } from "@tanstack/react-router";

import { PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, StickyTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TabsTrigger } from "@components/ui/tabs";
import {
  HostCertificatesCard,
  HostIdentityCard,
  HostInfoCard,
  HostLabelsCard,
  HostUsersCard,
} from "@features/hosts/components/host-detail-cards";
import { HostHeader } from "@features/hosts/components/host-header";
import { HostMunkiTab } from "@features/hosts/components/host-munki-tab";
import { HostOsqueryChecksTab } from "@features/hosts/components/host-osquery-checks-tab";
import { HostOsqueryReportsTab } from "@features/hosts/components/host-osquery-reports-tab";
import { HostSantaTab } from "@features/hosts/components/host-santa-tab";
import { HostSoftwareTab } from "@features/hosts/components/host-software-tab";
import { useHost, useHostMunkiState, useHostSantaState } from "@features/hosts/queries";

const hostSections = [
  { value: "details", label: "Details", path: "/hosts/$id" },
  { value: "software", label: "Software", path: "/hosts/$id/software" },
  { value: "reports", label: "Reports", path: "/hosts/$id/reports" },
  { value: "checks", label: "Checks", path: "/hosts/$id/checks" },
  { value: "munki", label: "Munki", path: "/hosts/$id/munki" },
  { value: "santa", label: "Santa", path: "/hosts/$id/santa" },
] as const;

export function HostDetailPage() {
  const hostID = useHostID();
  const query = useHost(hostID, { refetchInterval: 30_000 });
  const host = query.data;

  if (query.error || !host) {
    return (
      <QueryGate
        title="Failed to load host"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  return (
    <PageShell>
      <HostHeader host={host} />
      <HostSectionNav hostID={hostID} />
      <Outlet />
    </PageShell>
  );
}

export function HostDetailsPage() {
  const query = useHost(useHostID());
  const host = query.data;

  if (query.error || !host) {
    return (
      <QueryGate
        title="Failed to load host details"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <HostInfoCard host={host} />
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,28rem),1fr))] items-start gap-5">
        <HostIdentityCard host={host} />
        <HostLabelsCard host={host} />
        <HostUsersCard host={host} />
      </div>
      <HostCertificatesCard host={host} />
    </div>
  );
}

export function HostSoftwarePage() {
  return <HostSoftwareTab hostId={useHostID()} />;
}

export function HostReportsPage() {
  return <HostOsqueryReportsTab hostId={useHostID()} />;
}

export function HostChecksPage() {
  return <HostOsqueryChecksTab hostId={useHostID()} />;
}

export function HostMunkiPage() {
  const hostID = useHostID();
  const query = useHostMunkiState(hostID);
  return (
    <HostMunkiTab
      hostId={hostID}
      munki={query.data}
      stateError={query.error}
      onStateRetry={() => void query.refetch()}
    />
  );
}

export function HostSantaPage() {
  const hostID = useHostID();
  const query = useHostSantaState(hostID);
  return (
    <HostSantaTab
      hostId={hostID}
      santa={query.data}
      stateError={query.error}
      onStateRetry={() => void query.refetch()}
    />
  );
}

function HostSectionNav({ hostID }: { hostID: number }) {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const active =
    hostSections.find(
      (section) => section.value !== "details" && pathname.endsWith(`/${section.value}`),
    )?.value ?? "details";

  return (
    <ScrollableTabs value={active}>
      <StickyTabsList>
        {hostSections.map((section) => (
          <TabsTrigger
            key={section.value}
            value={section.value}
            render={<Link to={section.path} params={{ id: String(hostID) }} preload="intent" />}
            nativeButton={false}
          >
            {section.label}
          </TabsTrigger>
        ))}
      </StickyTabsList>
    </ScrollableTabs>
  );
}

function useHostID() {
  const { id } = useParams({ from: "/_authenticated/hosts/$id" });
  return Number(id);
}
