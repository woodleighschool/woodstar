import { useState } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import { Card, CardContent } from "@components/ui/card";
import type { ChartConfig } from "@components/ui/chart";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@components/ui/chart";
import type { OsqueryHostStatusPoint } from "@lib/api";

import { formatHistoryDateTime, formatHistoryTick, withHistoryGaps } from "./chart-data";
import { HistoryChartFrame } from "./chart-frame";
import { historyBounds, useHostStatusHistory, type HistoryRange } from "./queries";

const chartConfig = {
  online_percent: { label: "Online %", color: "var(--status-online)" },
} satisfies ChartConfig;

type HostStatusChartPoint = {
  bucket: string;
  online_count: number;
  online_percent: number | null;
  total_count: number;
};

type HostStatusChartDatum = Omit<HostStatusChartPoint, "bucket"> & { bucket: number };

const percentFormatter = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 1,
});

const hostStatusTooltipFormatter: NonNullable<
  React.ComponentProps<typeof ChartTooltipContent>["formatter"]
> = (_value, _name, item) => {
  const point: unknown = item.payload;
  if (!isHostStatusChartDatum(point)) return null;

  return (
    <div className="grid min-w-32 flex-1 gap-1.5">
      <div className="flex items-center justify-between gap-4">
        <span className="flex items-center gap-2 text-muted-foreground">
          <span className="size-2.5 rounded-[2px] bg-(--color-online_percent)" />
          Online %
        </span>
        <span className="font-medium text-foreground tabular-nums">
          {formatPercent(point.online_percent)}
        </span>
      </div>
      <div className="flex items-center justify-between gap-4">
        <span className="text-muted-foreground">Online</span>
        <span className="font-medium text-foreground tabular-nums">
          {point.online_count.toLocaleString()}
        </span>
      </div>
      <div className="flex items-center justify-between gap-4">
        <span className="text-muted-foreground">Total</span>
        <span className="font-medium text-foreground tabular-nums">
          {point.total_count.toLocaleString()}
        </span>
      </div>
    </div>
  );
};

const hostStatusTooltipLabelFormatter: NonNullable<
  React.ComponentProps<typeof ChartTooltipContent>["labelFormatter"]
> = (_label, payload) => {
  const point: unknown = payload[0]?.payload;
  return isHostStatusChartDatum(point) ? formatHistoryDateTime(point.bucket) : "-";
};

export function HostStatusChart() {
  const [range, setRange] = useState<HistoryRange>("24h");
  const history = useHostStatusHistory(range);
  const bounds = historyBounds(range, history.dataUpdatedAt);
  const points = history.data?.map(toHostStatusChartPoint);
  const latest = points?.at(-1);

  return (
    <Card>
      <CardContent>
        <HistoryChartFrame
          title="Hosts Online"
          description="Online means an osquery check-in within the last five minutes."
          summary={
            latest
              ? [
                  { value: latest.online_count.toLocaleString(), label: "online" },
                  { value: latest.total_count.toLocaleString(), label: "total" },
                ]
              : undefined
          }
          range={range}
          onRangeChange={setRange}
          error={history.error}
          errorTitle="Failed to load host history"
          onRetry={() => void history.refetch()}
          isLoading={history.isLoading}
          hasData={Boolean(points?.length && latest)}
          emptyTitle="No Host History Yet"
        >
          <ChartContainer config={chartConfig} className="aspect-auto h-72 w-full">
            <LineChart
              accessibilityLayer
              data={withHistoryGaps(points ?? [], bounds)}
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
              <YAxis
                domain={[0, 100]}
                ticks={[0, 25, 50, 75, 100]}
                tickLine={false}
                axisLine={false}
                width={44}
                tickFormatter={(value: number) => `${value}%`}
              />
              <ChartTooltip filterNull={false} content={<HostStatusTooltip />} />
              <Line
                dataKey="online_percent"
                type="monotone"
                stroke="var(--color-online_percent)"
                strokeWidth={2}
                dot={false}
                connectNulls={false}
                isAnimationActive={false}
              />
            </LineChart>
          </ChartContainer>
        </HistoryChartFrame>
      </CardContent>
    </Card>
  );
}

function toHostStatusChartPoint(point: OsqueryHostStatusPoint): HostStatusChartPoint {
  const totalCount = point.online_count + point.offline_count;
  return {
    bucket: point.bucket,
    online_count: point.online_count,
    online_percent: totalCount > 0 ? (point.online_count / totalCount) * 100 : null,
    total_count: totalCount,
  };
}

function HostStatusTooltip(props: React.ComponentProps<typeof ChartTooltipContent>) {
  const point: unknown = props.payload?.[0]?.payload;
  if (!isHostStatusChartDatum(point)) return null;

  return (
    <ChartTooltipContent
      {...props}
      labelFormatter={hostStatusTooltipLabelFormatter}
      formatter={hostStatusTooltipFormatter}
    />
  );
}

function isHostStatusChartDatum(value: unknown): value is HostStatusChartDatum {
  if (typeof value !== "object" || value === null) return false;
  return (
    "bucket" in value &&
    typeof value.bucket === "number" &&
    "online_count" in value &&
    typeof value.online_count === "number" &&
    "online_percent" in value &&
    (typeof value.online_percent === "number" || value.online_percent === null) &&
    "total_count" in value &&
    typeof value.total_count === "number"
  );
}

function formatPercent(value: number | null): string {
  return value === null ? "-" : `${percentFormatter.format(value)}%`;
}
