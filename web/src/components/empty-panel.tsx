import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function EmptyPanel({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        `
          flex min-h-32 w-full items-center justify-center rounded-lg border
          border-dashed px-6 text-center text-sm text-muted-foreground
        `,
        className,
      )}
    >
      {children}
    </div>
  );
}
