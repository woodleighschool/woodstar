import { getRouteApi } from "@tanstack/react-router";
import { addDays, parseISO, startOfDay } from "date-fns";
import { useMemo } from "react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { countLabel } from "@lib/utils";

import { type ActivityFilterState, ActivityFilters } from "./filters";
import { ActivityList } from "./list";
import { useActivity } from "./queries";

const routeApi = getRouteApi("/_authenticated/activity");

export function ActivityPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const bounds = useMemo(() => activityBounds(search.from, search.to), [search.from, search.to]);
  const activity = useActivity({
    q: search.q,
    page: search.page,
    per_page: search.per_page,
    area: search.area,
    actor_kind: search.scope === "all" ? undefined : search.scope,
    action: search.action,
    since: bounds.since,
    before: bounds.before,
  });
  const filterState: ActivityFilterState = {
    q: search.q,
    scope: search.scope,
    area: search.area,
    action: search.action,
    from: search.from,
    to: search.to,
  };
  const filtersChanged =
    search.q !== undefined ||
    search.scope !== "user" ||
    search.area !== undefined ||
    search.action !== undefined ||
    search.from !== undefined ||
    search.to !== undefined;

  function updateFilters(next: Partial<ActivityFilterState>) {
    void navigate({
      replace: true,
      search: (previous) => ({ ...previous, ...next, page: 1 }),
    });
  }

  return (
    <PageShell>
      <PageHeader
        title="Activity"
        description="Review administrator actions and retained agent activity."
        meta={activity.data ? countLabel(activity.data.count, "event") : undefined}
      />
      <ActivityFilters
        value={filterState}
        loading={activity.isFetching}
        showReset={filtersChanged}
        onChange={updateFilters}
        onReset={() =>
          void navigate({
            replace: true,
            search: { page: 1, per_page: search.per_page, scope: "user" },
          })
        }
      />
      <ActivityList
        data={activity.data}
        error={activity.error}
        isLoading={activity.isLoading}
        isFetching={activity.isFetching}
        page={search.page}
        perPage={search.per_page}
        showArea
        emptyTitle={filtersChanged ? "No Matching Activity" : "No Activity Yet"}
        emptyDescription={
          filtersChanged
            ? "Try changing or resetting the current filters."
            : "Administrator actions will appear here."
        }
        onPageChange={(page) =>
          void navigate({
            replace: true,
            search: (previous) => ({ ...previous, page }),
          })
        }
        onRetry={() => void activity.refetch()}
      />
    </PageShell>
  );
}

function activityBounds(
  fromValue: string | undefined,
  toValue: string | undefined,
): { since?: string; before?: string } {
  return {
    since: fromValue ? startOfDay(parseISO(fromValue)).toISOString() : undefined,
    before: toValue ? addDays(startOfDay(parseISO(toValue)), 1).toISOString() : undefined,
  };
}
