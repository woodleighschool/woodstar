import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@components/ui/empty";
import { Skeleton } from "@components/ui/skeleton";

import { useActivity } from "./queries";
import { ActivityTimeline } from "./timeline";

export function RecentOsqueryActivityCard() {
  const activity = useActivity({ area: "osquery", page: 1, per_page: 8 });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Activity</CardTitle>
        <CardDescription>Enrollments and administrator actions related to osquery.</CardDescription>
        <CardAction>
          <Button
            variant="outline"
            size="sm"
            render={<Link to="/activity" search={{ scope: "all", area: "osquery" }} />}
            nativeButton={false}
          >
            View All
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {activity.error ? (
          <QueryError
            title="Failed to load recent activity"
            error={activity.error}
            onRetry={() => void activity.refetch()}
          />
        ) : activity.isLoading ? (
          <div className="space-y-4">
            {Array.from({ length: 4 }, (_, index) => (
              <Skeleton key={index} className="h-11 w-full" />
            ))}
          </div>
        ) : activity.data?.items.length ? (
          <ActivityTimeline events={activity.data.items} />
        ) : (
          <Empty className="min-h-40">
            <EmptyHeader>
              <EmptyTitle>No osquery Activity Yet</EmptyTitle>
              <EmptyDescription>
                Enrollments and administrative changes will appear here.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
      </CardContent>
    </Card>
  );
}
