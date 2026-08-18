import { HISTORY_REFRESH_MS, type HistoryRange } from "./queries";

type TimestampedPoint<Point> = Omit<Point, "bucket"> & { bucket: number };

export function withHistoryGaps<Point extends { bucket: string }>(
  points: Point[],
  bounds: [number, number],
): Array<TimestampedPoint<Point> | { bucket: number }> {
  const data: Array<TimestampedPoint<Point> | { bucket: number }> = [];
  const timestamped = points.map((point) => ({
    ...point,
    bucket: new Date(point.bucket).getTime(),
  }));
  const first = timestamped[0];
  if (first && first.bucket - bounds[0] > HISTORY_REFRESH_MS * 1.5) {
    data.push({ bucket: bounds[0] });
  }
  for (const [index, point] of timestamped.entries()) {
    const previous = timestamped[index - 1];
    if (previous && point.bucket - previous.bucket > HISTORY_REFRESH_MS * 1.5) {
      data.push({ bucket: previous.bucket + HISTORY_REFRESH_MS });
    }
    data.push(point);
  }
  const last = timestamped.at(-1);
  if (last && bounds[1] - last.bucket > HISTORY_REFRESH_MS * 1.5) {
    data.push({ bucket: bounds[1] });
  }
  return data;
}

export function formatHistoryTick(value: number, range: HistoryRange): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(undefined, {
    month: range === "24h" ? undefined : "short",
    day: range === "24h" ? undefined : "numeric",
    hour: range === "24h" ? "numeric" : undefined,
  }).format(date);
}
