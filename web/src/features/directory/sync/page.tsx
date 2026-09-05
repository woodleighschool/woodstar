import { RefreshCw, UserRound, UsersRound } from "lucide-react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { ResourceOverviewCard } from "@components/resource-overview-card";
import { Button } from "@components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import { Can, useCan } from "@features/authz/access";
import { useGroups } from "@features/directory/groups/queries";
import { useDirectorySync, useTriggerDirectorySync } from "@features/directory/sync/queries";
import { useUsers } from "@features/directory/users/queries";
import { cn, formatRelative } from "@lib/utils";

const OVERVIEW_COUNT_PARAMS = { page: 1, per_page: 1 } as const;

export function DirectoryOverviewPage() {
  return (
    <PageShell>
      <PageHeader
        title="Directory"
        description="Manage synced and local identities."
        actions={<DirectorySyncAction />}
      />

      <div className="grid min-w-0 gap-4 md:grid-cols-3">
        <Can resource="users" access="view">
          <UsersOverviewCard />
        </Can>
        <Can resource="groups" access="view">
          <GroupsOverviewCard />
        </Can>
      </div>
    </PageShell>
  );
}

function UsersOverviewCard() {
  const users = useUsers(OVERVIEW_COUNT_PARAMS);
  return (
    <Link to="/directory/users">
      <ResourceOverviewCard
        title="Users"
        count={users.data?.count}
        loading={users.isLoading}
        error={users.error}
        icon={UserRound}
      />
    </Link>
  );
}

function GroupsOverviewCard() {
  const groups = useGroups(OVERVIEW_COUNT_PARAMS);
  return (
    <Link to="/directory/groups">
      <ResourceOverviewCard
        title="Groups"
        count={groups.data?.count}
        loading={groups.isLoading}
        error={groups.error}
        icon={UsersRound}
      />
    </Link>
  );
}

function DirectorySyncAction() {
  const canEdit = useCan({ resource: "directory", access: "edit" });
  const sync = useDirectorySync();
  const trigger = useTriggerDirectorySync();
  const status = sync.data;
  const active = status?.activity === "queued" || status?.activity === "running";
  const syncing = active || trigger.isPending;
  const lastRun = status?.activity === "idle" && !syncing ? status.last_run : undefined;
  const lastRunAt = lastRun?.finished_at ?? lastRun?.started_at ?? lastRun?.queued_at;

  if (!canEdit) return null;
  if (status && !status.enabled) return null;

  return (
    <>
      {lastRun?.outcome === "failed" ? <DirectorySyncFailure error={lastRun.error} /> : null}
      <DirectorySyncButton
        disabled={!status || syncing}
        syncing={syncing}
        lastSyncedAt={lastRun?.outcome === "succeeded" ? lastRunAt : undefined}
        onClick={() => trigger.mutate()}
      />
    </>
  );
}

function DirectorySyncButton({
  disabled,
  syncing,
  lastSyncedAt,
  onClick,
}: {
  disabled: boolean;
  syncing: boolean;
  lastSyncedAt?: string;
  onClick: () => void;
}) {
  const button = (
    <Button size="sm" disabled={disabled} onClick={onClick}>
      <RefreshCw className={cn(syncing && "animate-spin")} data-icon="inline-start" />
      Sync IdP
    </Button>
  );

  if (!lastSyncedAt) return button;

  return (
    <Tooltip>
      <TooltipTrigger render={button} />
      <TooltipContent side="bottom" align="end">
        Last synced {formatRelative(lastSyncedAt)}
      </TooltipContent>
    </Tooltip>
  );
}

function DirectorySyncFailure({ error }: { error?: string }) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            aria-label="Directory sync status"
            className="cursor-default text-sm text-muted-foreground underline decoration-dotted underline-offset-4"
          />
        }
      >
        Last sync failed
      </TooltipTrigger>
      <TooltipContent side="bottom" align="end" className="max-w-sm text-left whitespace-normal">
        {error || "The last sync failed."}
      </TooltipContent>
    </Tooltip>
  );
}
