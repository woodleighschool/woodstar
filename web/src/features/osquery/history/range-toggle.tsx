import { ToggleGroup, ToggleGroupItem } from "@components/ui/toggle-group";
import { isOneOf } from "@lib/utils";

import type { HistoryRange } from "./queries";

const ranges = ["24h", "7d", "30d"] as const satisfies readonly HistoryRange[];

export function HistoryRangeToggle({
  value,
  onChange,
}: {
  value: HistoryRange;
  onChange: (value: HistoryRange) => void;
}) {
  return (
    <ToggleGroup
      value={[value]}
      onValueChange={(values) => {
        const next = values.at(-1);
        if (isOneOf(next, ranges)) onChange(next);
      }}
      variant="outline"
      size="sm"
      spacing={0}
      aria-label="History range"
    >
      {ranges.map((range) => (
        <ToggleGroupItem key={range} value={range} aria-label={`Show ${range} of history`}>
          {range}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  );
}
