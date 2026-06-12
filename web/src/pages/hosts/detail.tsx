import { useParams } from "@tanstack/react-router";

import { HostChecksTab } from "@/components/hosts/host-checks-tab";
import {
  HostCertificatesCard,
  HostIdentityCard,
  HostInfoCard,
  HostLabelsCard,
  HostUsersCard,
} from "@/components/hosts/host-detail-cards";
import { HostHeader } from "@/components/hosts/host-header";
import { HostMunkiTab } from "@/components/hosts/host-munki-tab";
import { HostReportsTab } from "@/components/hosts/host-reports-tab";
import { HostSantaTab } from "@/components/hosts/host-santa-tab";
import { HostSoftwareTab } from "@/components/hosts/host-software-tab";
import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useHost } from "@/hooks/use-hosts";

export function HostDetailPage() {
  const { hostId } = useParams({ from: "/_authenticated/hosts/$hostId" });
  const hostID = Number(hostId);
  const query = useHost(hostID);
  const host = query.data;

  if (query.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load host"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      </PageShell>
    );
  }

  if (!host) {
    return null;
  }

  return (
    <PageShell className="gap-6">
      <HostHeader host={host} />

      <Tabs defaultValue="details">
        <TabsList>
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="software">Software</TabsTrigger>
          <TabsTrigger value="reports">Reports</TabsTrigger>
          <TabsTrigger value="checks">Checks</TabsTrigger>
          {host.munki ? <TabsTrigger value="munki">Munki</TabsTrigger> : null}
          {host.santa ? <TabsTrigger value="santa">Santa</TabsTrigger> : null}
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
          <HostReportsTab hostId={hostID} />
        </TabsContent>

        <TabsContent value="checks">
          <HostChecksTab hostId={hostID} />
        </TabsContent>

        {host.munki ? (
          <TabsContent value="munki">
            <HostMunkiTab host={host} />
          </TabsContent>
        ) : null}

        {host.santa ? (
          <TabsContent value="santa">
            <HostSantaTab hostId={hostID} host={host} />
          </TabsContent>
        ) : null}
      </Tabs>
    </PageShell>
  );
}
