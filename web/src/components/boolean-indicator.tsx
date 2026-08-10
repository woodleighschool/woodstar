import { Check, X } from "lucide-react";

import { cn } from "@lib/utils";

export function BooleanIndicator({
  value,
  trueLabel = "Yes",
  falseLabel = "No",
  tone = "neutral",
  className,
}: {
  value: boolean;
  trueLabel?: string;
  falseLabel?: string;
  tone?: "neutral" | "positive";
  className?: string;
}) {
  const Icon = value ? Check : X;
  return (
    <span
      className={cn(
        "inline-flex items-center",
        tone === "positive" && (value ? "text-status-online" : "text-muted-foreground"),
        className,
      )}
    >
      <Icon className="size-4" aria-hidden="true" />
      <span className="sr-only">{value ? trueLabel : falseLabel}</span>
    </span>
  );
}
