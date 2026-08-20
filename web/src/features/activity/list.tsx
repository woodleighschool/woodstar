import { useEffect } from "react";

import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@components/ui/empty";
import { Separator } from "@components/ui/separator";
import { Skeleton } from "@components/ui/skeleton";
import type { ApiError, PageActivityEvent } from "@lib/api";

import { ActivityTimeline } from "./timeline";

export function ActivityList({
  data,
  error,
  isLoading,
  isFetching,
  page,
  perPage,
  showArea = false,
  emptyTitle,
  emptyDescription,
  onPageChange,
  onRetry,
}: {
  data?: PageActivityEvent;
  error: ApiError | null;
  isLoading: boolean;
  isFetching: boolean;
  page: number;
  perPage: number;
  showArea?: boolean;
  emptyTitle: string;
  emptyDescription: string;
  onPageChange: (page: number) => void;
  onRetry: () => void;
}) {
  const pages = Math.max(1, Math.ceil((data?.count ?? 0) / perPage));

  useEffect(() => {
    if (data && page > pages) onPageChange(pages);
  }, [data, onPageChange, page, pages]);

  if (error) {
    return <QueryError title="Failed to Load Activity" error={error} onRetry={onRetry} />;
  }
  if (isLoading) {
    return (
      <div className="flex flex-col gap-5">
        {Array.from({ length: 8 }, (_, index) => (
          <Skeleton key={index} className="h-12 w-full" />
        ))}
      </div>
    );
  }
  if (!data?.items.length) {
    return (
      <Empty className="min-h-64 border">
        <EmptyHeader>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          <EmptyDescription>{emptyDescription}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="flex flex-col gap-4" aria-busy={isFetching}>
      <ActivityTimeline events={data.items} showArea={showArea} />
      {pages > 1 ? (
        <>
          <Separator />
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm text-muted-foreground">
              Page {page} of {pages}
            </span>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page <= 1 || isFetching}
                onClick={() => onPageChange(Math.max(1, page - 1))}
              >
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page >= pages || isFetching}
                onClick={() => onPageChange(Math.min(pages, page + 1))}
              >
                Next
              </Button>
            </div>
          </div>
        </>
      ) : null}
    </div>
  );
}
