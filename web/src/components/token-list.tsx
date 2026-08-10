import type { ReactNode } from "react";

import { Badge } from "@components/ui/badge";
import { cn } from "@lib/utils";

export function TokenList({
  values,
  empty = <span className="text-muted-foreground">-</span>,
  className,
}: {
  values: readonly string[];
  empty?: ReactNode;
  className?: string;
}) {
  if (values.length === 0) return empty;

  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {values.map((value, index) => (
        <Badge key={`${value}:${index}`} variant="secondary" className="font-normal">
          {value}
        </Badge>
      ))}
    </div>
  );
}
