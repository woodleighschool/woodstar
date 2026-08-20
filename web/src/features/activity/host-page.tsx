import { getRouteApi } from "@tanstack/react-router";

import { ActivityList } from "./list";
import { useActivity } from "./queries";

const routeApi = getRouteApi("/_authenticated/hosts/$id/activity");

export function HostActivityPage() {
  const { id } = routeApi.useParams();
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const activity = useActivity({
    page: search.page,
    per_page: search.per_page,
    subject_type: "host",
    subject_id: Number(id),
  });

  return (
    <ActivityList
      data={activity.data}
      error={activity.error}
      isLoading={activity.isLoading}
      isFetching={activity.isFetching}
      page={search.page}
      perPage={search.per_page}
      showArea
      emptyTitle="No Host Activity Yet"
      emptyDescription="Actions and enrollments associated with this host will appear here."
      onPageChange={(page) =>
        void navigate({ replace: true, search: (previous) => ({ ...previous, page }) })
      }
      onRetry={() => void activity.refetch()}
    />
  );
}
