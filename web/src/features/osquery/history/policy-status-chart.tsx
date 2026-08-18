import { useState } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import { QueryError } from "@components/query-error";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@components/ui/card";
import type { ChartConfig } from "@components/ui/chart";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@components/ui/chart";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@components/ui/empty";
import { Skeleton } from "@components/ui/skeleton";
import { formatDateTime } from "@lib/utils";

import { formatHistoryTick, withHistoryGaps } from "./chart-data";
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
  const history = usePolicyStatusHistory(policyID, range);
  const bounds = historyBounds(range, history.dataUpdatedAt || Date.now());

  return (
    <Card>
      <CardHeader>
        <CardTitle>Status history</CardTitle>
        <CardDescription>
          Five-minute snapshots of this policy across assigned hosts.
        </CardDescription>
        <CardAction>
          <HistoryRangeToggle value={range} onChange={setRange} />
        </CardAction>
      </CardHeader>
      <CardContent>
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
                    labelFormatter={(_, payload) => formatDateTime(payload[0]?.payload.bucket)}
                  />
                }
              />
              <ChartLegend content={<ChartLegendContent />} />
              {Object.keys(chartConfig).map((key) => (
                <Line
                  key={key}
                  dataKey={key}
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
              <EmptyTitle>No policy history yet</EmptyTitle>
              <EmptyDescription>
                The first five-minute snapshot will appear here shortly.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
      </CardContent>
    </Card>
  );
}
