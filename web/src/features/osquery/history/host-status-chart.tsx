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

import { formatHistoryDateTime, formatHistoryTick, withHistoryGaps } from "./chart-data";
import { historyBounds, useHostStatusHistory, type HistoryRange } from "./queries";
import { HistoryRangeToggle } from "./range-toggle";

const chartConfig = {
  online_count: { label: "Online", color: "var(--status-online)" },
  offline_count: { label: "Offline", color: "var(--status-offline)" },
} satisfies ChartConfig;

export function HostStatusChart() {
  const [range, setRange] = useState<HistoryRange>("24h");
  const [hiddenSeries, setHiddenSeries] = useState<string[]>([]);
  const history = useHostStatusHistory(range);
  const bounds = historyBounds(range, history.dataUpdatedAt || Date.now());
  const toggleSeries = (key: string) => {
    setHiddenSeries((current) =>
      current.includes(key) ? current.filter((value) => value !== key) : [...current, key],
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Hosts Online</CardTitle>
        <CardDescription>
          Online means an osquery check-in within the last five minutes.
        </CardDescription>
        <CardAction>
          <HistoryRangeToggle value={range} onChange={setRange} />
        </CardAction>
      </CardHeader>
      <CardContent>
        {history.error ? (
          <QueryError
            title="Failed to load host history"
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
                    labelFormatter={(_, payload) =>
                      formatHistoryDateTime(payload[0]?.payload.bucket)
                    }
                  />
                }
              />
              <ChartLegend
                content={
                  <ChartLegendContent hiddenKeys={hiddenSeries} onItemToggle={toggleSeries} />
                }
              />
              <Line
                dataKey="online_count"
                hide={hiddenSeries.includes("online_count")}
                type="monotone"
                stroke="var(--color-online_count)"
                strokeWidth={2}
                dot={false}
                connectNulls={false}
                isAnimationActive={false}
              />
              <Line
                dataKey="offline_count"
                hide={hiddenSeries.includes("offline_count")}
                type="monotone"
                stroke="var(--color-offline_count)"
                strokeWidth={2}
                dot={false}
                connectNulls={false}
                isAnimationActive={false}
              />
            </LineChart>
          </ChartContainer>
        ) : (
          <HistoryEmpty />
        )}
      </CardContent>
    </Card>
  );
}

function HistoryEmpty() {
  return (
    <Empty className="min-h-72">
      <EmptyHeader>
        <EmptyTitle>No Host History Yet</EmptyTitle>
        <EmptyDescription>
          The first five-minute snapshot will appear here shortly.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
