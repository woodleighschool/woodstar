import { Outlet, useNavigate, useParams, useRouterState } from "@tanstack/react-router";
import { ChevronDown, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";

import { PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import { TabsTrigger } from "@components/ui/tabs";
import { useAuth } from "@features/auth/queries";
import {
  HostCertificatesCard,
  HostIdentityCard,
  HostInfoCard,
  HostLabelsCard,
  HostUsersCard,
} from "@features/hosts/components/host-detail-cards";
import { HostHeader } from "@features/hosts/components/host-header";
import { HostHeartbeatTable } from "@features/hosts/components/host-heartbeats";
import { HostMunkiTab } from "@features/hosts/components/host-munki-tab";
import { HostOsqueryChecksTab } from "@features/hosts/components/host-osquery-checks-tab";
import { HostOsqueryReportsTab } from "@features/hosts/components/host-osquery-reports-tab";
import { HostSantaTab } from "@features/hosts/components/host-santa-tab";
import { HostSoftwareTab } from "@features/hosts/components/host-software-tab";
import { HostDeleteDialog } from "@features/hosts/delete-dialog";
import {
  useHost,
  useHostMunkiState,
  useHostSantaState,
  useRequestHostInventoryRefresh,
} from "@features/hosts/queries";
import type { HostDetail } from "@lib/api";

const hostSections = [
  { value: "details", label: "Details", path: "/hosts/$id" },
  { value: "software", label: "Software", path: "/hosts/$id/software" },
  { value: "reports", label: "Reports", path: "/hosts/$id/reports" },
  { value: "checks", label: "Checks", path: "/hosts/$id/checks" },
  {
    value: "munki",
    label: "Munki",
    path: "/hosts/$id/munki",
    heartbeatSource: "munki",
  },
  {
    value: "santa",
    label: "Santa",
    path: "/hosts/$id/santa",
    heartbeatSource: "santa",
  },
] as const;

export function HostDetailPage() {
  const hostID = useHostID();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const refresh = useRequestHostInventoryRefresh();
  const query = useHost(hostID, {
    refetchInterval: (hostQuery) =>
      hostQuery.state.data?.inventory_refresh_requested ? 2_000 : 30_000,
  });
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
    <>
      <PageShell>
        <HostHeader
          host={host}
          actions={
            user?.role === "admin" ? (
              <DropdownMenu>
                <DropdownMenuTrigger render={<Button type="button" variant="outline" size="sm" />}>
                  Actions
                  <ChevronDown data-icon="inline-end" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="min-w-44">
                  <DropdownMenuGroup>
                    <DropdownMenuItem
                      disabled={refresh.isPending || host.inventory_refresh_requested}
                      onClick={() => refresh.mutate(host.id)}
                    >
                      <RefreshCw />
                      {refresh.isPending
                        ? "Requesting…"
                        : host.inventory_refresh_requested
                          ? "Refresh requested"
                          : "Refresh inventory"}
                    </DropdownMenuItem>
                    <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
                      <Trash2 />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null
          }
        />
        <HostSectionNav hostID={hostID} host={host} />
        <Outlet />
      </PageShell>
      <HostDeleteDialog
        host={host}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/hosts" })}
      />
    </>
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
      <HostHeartbeatTable heartbeats={host.heartbeats} />
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
  const host = useHost(hostID).data;
  const query = useHostMunkiState(hostID);
  return (
    <HostMunkiTab
      hostId={hostID}
      hostSerial={host?.hardware.serial}
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

function HostSectionNav({ hostID, host }: { hostID: number; host: HostDetail }) {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const active =
    hostSections.find(
      (section) => section.value !== "details" && pathname.endsWith(`/${section.value}`),
    )?.value ?? "details";

  return (
    <ScrollableTabs value={active}>
      <ScrollableTabsList>
        {hostSections
          .filter(
            (section) =>
              !("heartbeatSource" in section) ||
              host.heartbeats.some((heartbeat) => heartbeat.source === section.heartbeatSource),
          )
          .map((section) => (
            <TabsTrigger
              key={section.value}
              value={section.value}
              render={<Link to={section.path} params={{ id: String(hostID) }} preload="intent" />}
              nativeButton={false}
            >
              {section.label}
            </TabsTrigger>
          ))}
      </ScrollableTabsList>
    </ScrollableTabs>
  );
}

function useHostID() {
  const { id } = useParams({ from: "/_authenticated/hosts/$id" });
  return Number(id);
}
