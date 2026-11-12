import type { ReactNode } from "react";

import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@components/ui/empty";

// Empty state for a first-class resource list. Nested tables, tabs, and detail
// sections use text-only PanelEmptyState instead.
export function DataTableEmpty({
  icon,
  title,
  description,
  filtered = false,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  filtered?: boolean;
}) {
  return (
    <Empty className="min-h-72 border-0">
      <EmptyHeader>
        <EmptyMedia variant="icon">{icon}</EmptyMedia>
        <EmptyTitle>{filtered ? "No Matches" : title}</EmptyTitle>
        {!filtered ? <EmptyDescription>{description}</EmptyDescription> : null}
      </EmptyHeader>
    </Empty>
  );
}
