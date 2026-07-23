import { useParams } from "@tanstack/react-router";

import {
  HostCertificatesCard,
  HostIdentityCard,
  HostInfoCard,
  HostLabelsCard,
  HostUsersCard,
} from "@/components/hosts/host-detail-cards";
import { HostHeader } from "@/components/hosts/host-header";
import { HostMunkiTab } from "@/components/hosts/host-munki-tab";
import { HostOsqueryChecksTab } from "@/components/hosts/host-osquery-checks-tab";
import { HostOsqueryReportsTab } from "@/components/hosts/host-osquery-reports-tab";
import { HostSantaTab } from "@/components/hosts/host-santa-tab";
import { HostSoftwareTab } from "@/components/hosts/host-software-tab";
import { PageShell } from "@/components/layout/page-layout";
import { QueryGate } from "@/components/query-gate";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useHost, useHostMunkiState, useHostSantaState } from "@/hooks/use-hosts";

export function HostDetailPage() {
  const { id: hostId } = useParams({ from: "/_authenticated/hosts/$id" });
  const hostID = Number(hostId);
  const query = useHost(hostID, { refetchInterval: 30_000 });
  const munkiQuery = useHostMunkiState(hostID);
  const santaQuery = useHostSantaState(hostID);
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

  const showMunkiTab =
    munkiQuery.data !== null && (munkiQuery.data !== undefined || munkiQuery.error);
  const showSantaTab =
    santaQuery.data !== null && (santaQuery.data !== undefined || santaQuery.error);

  return (
    <PageShell className="gap-6">
      <HostHeader host={host} />

      <Tabs defaultValue="details">
        <TabsList>
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="software">Software</TabsTrigger>
          <TabsTrigger value="reports">Reports</TabsTrigger>
          <TabsTrigger value="checks">Checks</TabsTrigger>
          {showMunkiTab ? <TabsTrigger value="munki">Munki</TabsTrigger> : null}
          {showSantaTab ? <TabsTrigger value="santa">Santa</TabsTrigger> : null}
        </TabsList>

        <TabsContent value="details">
          <div className="flex flex-col gap-4">
            <HostInfoCard host={host} />
            <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,28rem),1fr))] items-start gap-4">
              <HostIdentityCard host={host} />
              <HostLabelsCard host={host} />
              <HostUsersCard host={host} />
            </div>
            <HostCertificatesCard host={host} />
          </div>
        </TabsContent>

        <TabsContent value="software">
          <HostSoftwareTab hostId={hostID} />
        </TabsContent>

        <TabsContent value="reports">
          <HostOsqueryReportsTab hostId={hostID} />
        </TabsContent>

        <TabsContent value="checks">
          <HostOsqueryChecksTab hostId={hostID} />
        </TabsContent>

        {showMunkiTab ? (
          <TabsContent value="munki">
            {munkiQuery.error ? (
              <QueryGate
                title="Failed to load Munki state"
                error={munkiQuery.error}
                onRetry={() => void munkiQuery.refetch()}
              />
            ) : munkiQuery.data ? (
              <HostMunkiTab munki={munkiQuery.data} />
            ) : null}
          </TabsContent>
        ) : null}

        {showSantaTab ? (
          <TabsContent value="santa">
            {santaQuery.error ? (
              <QueryGate
                title="Failed to load Santa state"
                error={santaQuery.error}
                onRetry={() => void santaQuery.refetch()}
              />
            ) : santaQuery.data ? (
              <HostSantaTab hostId={hostID} santa={santaQuery.data} />
            ) : null}
          </TabsContent>
        ) : null}
      </Tabs>
    </PageShell>
  );
}
