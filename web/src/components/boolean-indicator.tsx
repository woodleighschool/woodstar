import { Check, X } from "lucide-react";

import { cn } from "@lib/utils";

export function BooleanIndicator({
  value,
  trueLabel = "Yes",
  falseLabel = "No",
  className,
}: {
  value: boolean;
  trueLabel?: string;
  falseLabel?: string;
  className?: string;
}) {
  const Icon = value ? Check : X;
  return (
    <span
      className={cn(
        "inline-flex items-center",
        value ? "text-status-online" : "text-destructive",
        className,
      )}
    >
      <Icon className="size-4" aria-hidden="true" />
      <span className="sr-only">{value ? trueLabel : falseLabel}</span>
    </span>
  );
}
