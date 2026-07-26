import type { ReactNode } from "react";

import { cn } from "@lib/utils";

export function PanelEmptyState({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        `
          flex min-h-24 w-full items-center justify-center bg-muted/20 px-6
          text-center text-sm text-muted-foreground
        `,
        className,
      )}
    >
      {children}
    </div>
  );
}
