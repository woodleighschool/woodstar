import { useState } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import type { ChartConfig } from "@components/ui/chart";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@components/ui/chart";
import type { OsqueryPolicyStatusPoint } from "@lib/api";

import { formatHistoryDateTime, formatHistoryTick, withHistoryGaps } from "./chart-data";
import { HistoryChartFrame } from "./chart-frame";
import { historyBounds, usePolicyStatusHistory, type HistoryRange } from "./queries";

const chartConfig = {
  non_pass_percent: { label: "Non-pass rate", color: "var(--destructive)" },
  pass_count: { label: "Pass", color: "var(--status-online)" },
  fail_count: { label: "Fail", color: "var(--destructive)" },
  error_count: { label: "Error", color: "var(--warning)" },
  pending_count: { label: "Pending", color: "var(--status-offline)" },
} satisfies ChartConfig;

type PolicyStatusChartPoint = OsqueryPolicyStatusPoint & {
  assigned_count: number;
  non_pass_count: number;
  non_pass_percent: number | null;
  passing_percent: number | null;
};

type PolicyStatusChartDatum = Omit<PolicyStatusChartPoint, "bucket"> & { bucket: number };

const policyPercentFormatter = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

const policyStatuses = [
  { key: "pass_count", label: "Pass", colorClass: "bg-(--color-pass_count)" },
  { key: "fail_count", label: "Fail", colorClass: "bg-(--color-fail_count)" },
  { key: "error_count", label: "Error", colorClass: "bg-(--color-error_count)" },
  { key: "pending_count", label: "Pending", colorClass: "bg-(--color-pending_count)" },
] as const;

const policyStatusTooltipFormatter: NonNullable<
  React.ComponentProps<typeof ChartTooltipContent>["formatter"]
> = (_value, _name, item) => {
  const point: unknown = item.payload;
  if (!isPolicyStatusChartDatum(point)) return null;

  return (
    <div className="grid min-w-48 flex-1 grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-x-4 gap-y-1.5">
      {policyStatuses.map((status) => (
        <div key={status.key} className="col-span-3 grid grid-cols-subgrid items-center">
          <span className="flex items-center gap-2 text-muted-foreground">
            <span className={`size-2.5 rounded-[2px] ${status.colorClass}`} />
            {status.label}
          </span>
          <span className="text-right font-medium text-foreground tabular-nums">
            {point[status.key].toLocaleString()}
          </span>
          <span className="text-right font-medium text-foreground tabular-nums">
            {formatPolicyPercent(percentage(point[status.key], point.assigned_count))}
          </span>
        </div>
      ))}
    </div>
  );
};

const policyStatusTooltipLabelFormatter: NonNullable<
  React.ComponentProps<typeof ChartTooltipContent>["labelFormatter"]
> = (_label, payload) => {
  const point: unknown = payload[0]?.payload;
  return isPolicyStatusChartDatum(point) ? formatHistoryDateTime(point.bucket) : "-";
};

export function PolicyStatusChart({ policyID }: { policyID: number }) {
  const [range, setRange] = useState<HistoryRange>("24h");
  const history = usePolicyStatusHistory(policyID, range);
  const bounds = historyBounds(range, history.dataUpdatedAt || Date.now());
  const points = history.data?.map(toPolicyStatusChartPoint);
  const latest = points?.at(-1);

  return (
    <HistoryChartFrame
      title="Non-pass rate"
      description="Percentage of assigned hosts that are not passing."
      summary={
        latest
          ? [
              { value: formatPolicyPercent(latest.passing_percent), label: "passing" },
              {
                value: `${latest.non_pass_count.toLocaleString()} of ${latest.assigned_count.toLocaleString()}`,
                label: "hosts not passing",
              },
            ]
          : undefined
      }
      range={range}
      onRangeChange={setRange}
      error={history.error}
      errorTitle="Failed to load policy history"
      onRetry={() => void history.refetch()}
      isLoading={history.isLoading}
      hasData={Boolean(points?.length)}
      emptyTitle="No Policy History Yet"
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
            domain={[0, nonPassDomainMaximum]}
            tickLine={false}
            axisLine={false}
            width={44}
            tickFormatter={(value: number) => `${value.toLocaleString()}%`}
          />
          <ChartTooltip filterNull={false} content={<PolicyStatusTooltip />} />
          <Line
            dataKey="non_pass_percent"
            type="monotone"
            stroke="var(--color-non_pass_percent)"
            strokeWidth={2}
            dot={false}
            connectNulls={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ChartContainer>
    </HistoryChartFrame>
  );
}

function toPolicyStatusChartPoint(point: OsqueryPolicyStatusPoint): PolicyStatusChartPoint {
  const nonPassCount = point.fail_count + point.error_count + point.pending_count;
  const assignedCount = point.pass_count + nonPassCount;
  return {
    ...point,
    assigned_count: assignedCount,
    non_pass_count: nonPassCount,
    non_pass_percent: percentage(nonPassCount, assignedCount),
    passing_percent: percentage(point.pass_count, assignedCount),
  };
}

function PolicyStatusTooltip(props: React.ComponentProps<typeof ChartTooltipContent>) {
  const point: unknown = props.payload?.[0]?.payload;
  if (!isPolicyStatusChartDatum(point)) return null;

  return (
    <ChartTooltipContent
      {...props}
      labelFormatter={policyStatusTooltipLabelFormatter}
      formatter={policyStatusTooltipFormatter}
    />
  );
}

function isPolicyStatusChartDatum(value: unknown): value is PolicyStatusChartDatum {
  if (typeof value !== "object" || value === null) return false;
  return (
    "bucket" in value &&
    typeof value.bucket === "number" &&
    "pass_count" in value &&
    typeof value.pass_count === "number" &&
    "fail_count" in value &&
    typeof value.fail_count === "number" &&
    "error_count" in value &&
    typeof value.error_count === "number" &&
    "pending_count" in value &&
    typeof value.pending_count === "number" &&
    "assigned_count" in value &&
    typeof value.assigned_count === "number" &&
    "non_pass_percent" in value &&
    (typeof value.non_pass_percent === "number" || value.non_pass_percent === null)
  );
}

function percentage(count: number, total: number): number | null {
  return total > 0 ? (count / total) * 100 : null;
}

function formatPolicyPercent(value: number | null): string {
  return value === null ? "-" : `${policyPercentFormatter.format(value)}%`;
}

function nonPassDomainMaximum(dataMax: number): number {
  return Math.max(1, Math.ceil(dataMax));
}
