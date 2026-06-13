import type { ReactNode } from "react";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

// Empty state for a first-class resource list. Nested tables, tabs, and detail
// sections use text-only EmptyPanel instead.
export function DataTableEmpty({
  icon,
  title,
  description,
  filtered = false,
  filteredTitle = "No matches",
  filteredDescription,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  filtered?: boolean;
  filteredTitle?: string;
  filteredDescription: string;
}) {
  return (
    <Empty className="min-h-72 border-0">
      <EmptyHeader>
        <EmptyMedia variant="icon">{icon}</EmptyMedia>
        <EmptyTitle>{filtered ? filteredTitle : title}</EmptyTitle>
        <EmptyDescription>{filtered ? filteredDescription : description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
