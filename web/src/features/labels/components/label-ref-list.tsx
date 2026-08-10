import { useMemo } from "react";

import { encodeSort } from "@components/data-table/use-data-table-search";
import { Link } from "@components/link";
import { Badge } from "@components/ui/badge";
import { useLabels } from "@features/labels/queries";
import { MAX_PAGE_SIZE } from "@lib/pagination";

export function useLabelNameMap() {
  const labels = useLabels({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
  });

  return useMemo(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label.name])),
    [labels.data?.items],
  );
}

export function LabelRefList({ labelIDs }: { labelIDs: readonly number[] }) {
  const labelsByID = useLabelNameMap();

  if (labelIDs.length === 0) {
    return <span className="text-muted-foreground">-</span>;
  }

  return (
    <div className="flex flex-wrap gap-1.5">
      {labelIDs.map((labelID) => (
        <Badge
          key={labelID}
          variant="outline"
          className="font-normal"
          render={<Link to="/labels/$id" params={{ id: String(labelID) }} />}
        >
          {labelsByID.get(labelID) ?? `Label ${labelID}`}
        </Badge>
      ))}
    </div>
  );
}
