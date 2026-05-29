import { useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { HostChecksTab } from "@/components/hosts/host-checks-tab";
import {
  HostCertificatesCard,
  HostIdentityCard,
  HostInfoCard,
  HostLabelsCard,
  HostUsersCard,
} from "@/components/hosts/host-detail-cards";
import { HostHeader } from "@/components/hosts/host-header";
import { HostReportsTab } from "@/components/hosts/host-reports-tab";
import { HostSantaTab } from "@/components/hosts/host-santa-tab";
import { HostSoftwareTab } from "@/components/hosts/host-software-tab";
import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
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
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Host</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      </PageShell>
    );
  }

  if (query.isLoading || !host) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </PageShell>
    );
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

        {host.santa ? (
          <TabsContent value="santa">
            <HostSantaTab hostId={hostID} host={host} />
          </TabsContent>
        ) : null}
      </Tabs>
    </PageShell>
  );
}
