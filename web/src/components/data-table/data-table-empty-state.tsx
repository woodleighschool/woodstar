import type { ReactNode } from "react";

import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";

interface DataTableEmptyStateProps {
  icon: ReactNode;
  title: ReactNode;
  description: ReactNode;
}

export function DataTableEmptyState({ icon, title, description }: DataTableEmptyStateProps) {
  return (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant="icon">{icon}</EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
