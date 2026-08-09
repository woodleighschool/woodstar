import { RefreshCw, UserRound, UsersRound, type LucideIcon } from "lucide-react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { RelativeTime } from "@components/relative-time";
import { Button } from "@components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { Skeleton } from "@components/ui/skeleton";
import { useAuth } from "@features/auth/queries";
import { useGroups } from "@features/directory/groups/queries";
import { useDirectorySync, useTriggerDirectorySync } from "@features/directory/sync/queries";
import { useUsers } from "@features/directory/users/queries";
import type { SyncRun, SyncStatus } from "@lib/api";

const OVERVIEW_COUNT_PARAMS = { page: 1, per_page: 1 } as const;

export function DirectoryOverviewPage() {
  const users = useUsers(OVERVIEW_COUNT_PARAMS);
  const groups = useGroups(OVERVIEW_COUNT_PARAMS);

  return (
    <PageShell>
      <PageHeader title="Directory" description="Manage synced and local identities." />

      <div className="grid min-w-0 gap-4 md:grid-cols-3">
        <DirectoryResourceCard
          title="Users"
          count={users.data?.count}
          loading={users.isLoading}
          error={users.error}
          icon={UserRound}
          to="/directory/users"
        />
        <DirectoryResourceCard
          title="Groups"
          count={groups.data?.count}
          loading={groups.isLoading}
          error={groups.error}
          icon={UsersRound}
          to="/directory/groups"
        />
        <DirectorySyncCard />
      </div>
    </PageShell>
  );
}

function DirectoryResourceCard({
  title,
  count,
  loading,
  error,
  icon: Icon,
  to,
}: {
  title: string;
  count?: number;
  loading: boolean;
  error: { message?: string } | null;
  icon: LucideIcon;
  to: "/directory/users" | "/directory/groups";
}) {
  return (
    <Link
      to={to}
      data-slot="card-link"
      className="min-w-0 rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <Card size="sm" className="h-full min-w-0 transition-colors hover:bg-muted/50">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardAction>
            <Icon aria-hidden="true" className="text-muted-foreground" />
          </CardAction>
        </CardHeader>
        <CardContent>
          {loading ? (
            <Skeleton className="h-9 w-24" />
          ) : error ? (
            <p className="text-sm text-destructive">Count unavailable</p>
          ) : (
            <span className="text-3xl font-semibold tracking-tight tabular-nums">
              {(count ?? 0).toLocaleString()}
            </span>
          )}
        </CardContent>
      </Card>
    </Link>
  );
}

function DirectorySyncCard() {
  const { user } = useAuth();
  const sync = useDirectorySync();
  const trigger = useTriggerDirectorySync();
  const status = sync.data;
  const active = status?.activity === "queued" || status?.activity === "running";
  const disabled = active || trigger.isPending;
  const actionLabel = trigger.isPending
    ? "Starting"
    : status?.activity === "queued"
      ? "Queued"
      : status?.activity === "running"
        ? "Syncing"
        : "Run now";

  return (
    <Card size="sm" className="min-w-0">
      <CardHeader>
        <CardTitle>Sync</CardTitle>
        <CardAction>
          {user?.role === "admin" && status?.enabled ? (
            <Button size="sm" disabled={disabled} onClick={() => trigger.mutate()}>
              <RefreshCw data-icon="inline-start" />
              {actionLabel}
            </Button>
          ) : null}
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {sync.error && !status ? (
          <QueryError
            title="Failed to load sync status"
            error={sync.error}
            onRetry={() => void sync.refetch()}
          />
        ) : !status ? (
          <SyncDetailsSkeleton />
        ) : !status.enabled ? (
          <p className="text-sm text-muted-foreground">Entra ID sync is not configured.</p>
        ) : (
          <SyncDetails status={status} />
        )}
      </CardContent>
    </Card>
  );
}

function SyncDetails({ status }: { status: SyncStatus }) {
  const lastRun = status.last_run;
  return (
    <div className="flex flex-col gap-3">
      <p className={lastRun?.outcome === "failed" ? "text-sm text-destructive" : "text-sm"}>
        {lastRun ? (
          lastRun.outcome === "succeeded" ? (
            <>
              Last synced <RunTime run={lastRun} />.
            </>
          ) : (
            <>
              Sync failed <RunTime run={lastRun} />.
            </>
          )
        ) : (
          "No sync has run yet."
        )}
      </p>
      {lastRun?.outcome === "failed" ? (
        <p className="text-sm text-destructive">
          {lastRun.error || "The last sync failed without an error message."}
        </p>
      ) : null}
    </div>
  );
}

function RunTime({ run }: { run: SyncRun }) {
  return <RelativeTime value={run.finished_at ?? run.started_at ?? run.queued_at} />;
}

function SyncDetailsSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      <Skeleton className="h-5 w-full" />
      <Skeleton className="h-5 w-4/5" />
    </div>
  );
}
