import type { ReactNode } from "react";

import { QueryError } from "@components/query-error";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@components/ui/empty";
import { Separator } from "@components/ui/separator";
import { Skeleton } from "@components/ui/skeleton";

import type { HistoryRange } from "./queries";
import { HistoryRangeToggle } from "./range-toggle";

type HistorySummaryItem = {
  label: string;
  value: string;
};

export function HistoryChartFrame({
  title,
  description,
  summary,
  range,
  onRangeChange,
  error,
  errorTitle,
  onRetry,
  isLoading,
  hasData,
  emptyTitle,
  children,
}: {
  title: string;
  description: string;
  summary?: readonly HistorySummaryItem[];
  range: HistoryRange;
  onRangeChange: (range: HistoryRange) => void;
  error: { message?: string } | null | undefined;
  errorTitle: string;
  onRetry: () => void;
  isLoading: boolean;
  hasData: boolean;
  emptyTitle: string;
  children: ReactNode;
}) {
  return (
    <section className="flex min-w-0 flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="text-base/snug font-medium text-foreground">{title}</h2>
          <p className="text-sm text-muted-foreground">{description}</p>
          {summary?.length ? (
            <p className="flex flex-wrap items-baseline gap-x-4 gap-y-1 text-sm">
              {summary.map((item) => (
                <span key={item.label} className="flex items-baseline gap-1">
                  <span className="font-medium text-foreground">{item.value}</span>
                  <span className="text-muted-foreground">{item.label}</span>
                </span>
              ))}
            </p>
          ) : null}
        </div>
        <div className="shrink-0">
          <HistoryRangeToggle value={range} onChange={onRangeChange} />
        </div>
      </div>
      <Separator />
      {error ? (
        <QueryError title={errorTitle} error={error} onRetry={onRetry} />
      ) : isLoading ? (
        <Skeleton className="h-72 w-full" />
      ) : hasData ? (
        children
      ) : (
        <Empty className="min-h-72">
          <EmptyHeader>
            <EmptyTitle>{emptyTitle}</EmptyTitle>
            <EmptyDescription>
              The first five-minute snapshot will appear here shortly.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </section>
  );
}
