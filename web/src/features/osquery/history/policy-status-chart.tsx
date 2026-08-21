import { useState } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import { QueryError } from "@components/query-error";
import type { ChartConfig } from "@components/ui/chart";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@components/ui/chart";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@components/ui/empty";
import { Separator } from "@components/ui/separator";
import { Skeleton } from "@components/ui/skeleton";

import { formatHistoryDateTime, formatHistoryTick, withHistoryGaps } from "./chart-data";
import { historyBounds, usePolicyStatusHistory, type HistoryRange } from "./queries";
import { HistoryRangeToggle } from "./range-toggle";

const chartConfig = {
  pass_count: { label: "Pass", color: "var(--status-online)" },
  fail_count: { label: "Fail", color: "var(--destructive)" },
  error_count: { label: "Error", color: "var(--warning)" },
  pending_count: { label: "Pending", color: "var(--status-offline)" },
} satisfies ChartConfig;

export function PolicyStatusChart({ policyID }: { policyID: number }) {
  const [range, setRange] = useState<HistoryRange>("24h");
  const [hiddenSeries, setHiddenSeries] = useState<string[]>([]);
  const history = usePolicyStatusHistory(policyID, range);
  const bounds = historyBounds(range, history.dataUpdatedAt || Date.now());
  const toggleSeries = (key: string) => {
    setHiddenSeries((current) =>
      current.includes(key) ? current.filter((value) => value !== key) : [...current, key],
    );
  };

  return (
    <section className="flex min-w-0 flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="text-base/snug font-medium text-foreground">Status History</h2>
          <p className="text-sm text-muted-foreground">
            Five-minute snapshots of this policy across assigned hosts.
          </p>
        </div>
        <div className="shrink-0">
          <HistoryRangeToggle value={range} onChange={setRange} />
        </div>
      </div>
      <Separator />
      {history.error ? (
        <QueryError
          title="Failed to load policy history"
          error={history.error}
          onRetry={() => void history.refetch()}
        />
      ) : history.isLoading ? (
        <Skeleton className="h-72 w-full" />
      ) : history.data?.length ? (
        <ChartContainer config={chartConfig} className="aspect-auto h-72 w-full">
          <LineChart
            accessibilityLayer
            data={withHistoryGaps(history.data, bounds)}
            margin={{ left: 4, right: 12 }}
          >
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="bucket"
              type="number"
              scale="time"
              domain={bounds}
              allowDataOverflow
              tickLine={false}
              axisLine={false}
              minTickGap={32}
              tickFormatter={(value: number) => formatHistoryTick(value, range)}
            />
            <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={36} />
            <ChartTooltip
              content={
                <ChartTooltipContent
                  labelFormatter={(_, payload) => formatHistoryDateTime(payload[0]?.payload.bucket)}
                />
              }
            />
            <ChartLegend
              content={<ChartLegendContent hiddenKeys={hiddenSeries} onItemToggle={toggleSeries} />}
            />
            {Object.keys(chartConfig).map((key) => (
              <Line
                key={key}
                dataKey={key}
                hide={hiddenSeries.includes(key)}
                type="monotone"
                stroke={`var(--color-${key})`}
                strokeWidth={2}
                dot={false}
                connectNulls={false}
                isAnimationActive={false}
              />
            ))}
          </LineChart>
        </ChartContainer>
      ) : (
        <Empty className="min-h-72">
          <EmptyHeader>
            <EmptyTitle>No Policy History Yet</EmptyTitle>
            <EmptyDescription>
              The first five-minute snapshot will appear here shortly.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </section>
  );
}
