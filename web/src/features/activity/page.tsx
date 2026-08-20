import { useState } from "react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@components/ui/empty";
import { Skeleton } from "@components/ui/skeleton";
import { countLabel } from "@lib/utils";

import { useActivity } from "./queries";
import { ActivityTimeline } from "./timeline";

const PAGE_SIZE = 50;

export function ActivityPage() {
  const [page, setPage] = useState(1);
  const activity = useActivity({ page, per_page: PAGE_SIZE });
  const count = activity.data?.count ?? 0;
  const pages = Math.max(1, Math.ceil(count / PAGE_SIZE));

  return (
    <PageShell>
      <PageHeader
        title="Activity"
        description="A simple record of administrator actions and agent enrollments."
        meta={activity.data ? countLabel(count, "event") : undefined}
      />

      {activity.error ? (
        <QueryError
          title="Failed to Load Activity"
          error={activity.error}
          onRetry={() => void activity.refetch()}
        />
      ) : activity.isLoading ? (
        <div className="space-y-5">
          {Array.from({ length: 8 }, (_, index) => (
            <Skeleton key={index} className="h-12 w-full" />
          ))}
        </div>
      ) : activity.data?.items.length ? (
        <>
          <ActivityTimeline events={activity.data.items} showArea />
          {pages > 1 ? (
            <div className="flex items-center justify-between border-t pt-4">
              <span className="text-sm text-muted-foreground">
                Page {page} of {pages}
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= pages}
                  onClick={() => setPage((current) => Math.min(pages, current + 1))}
                >
                  Next
                </Button>
              </div>
            </div>
          ) : null}
        </>
      ) : (
        <Empty className="min-h-64 border">
          <EmptyHeader>
            <EmptyTitle>No Activity Yet</EmptyTitle>
            <EmptyDescription>
              Administrator actions and agent enrollments will appear here.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </PageShell>
  );
}
