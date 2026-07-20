import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function KeyValueGrid({
  className,
  children,
}: {
  className?: string;
  children?: ReactNode;
}) {
  return (
    <dl
      className={cn(
        "grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5",
        className,
      )}
    >
      {children}
    </dl>
  );
}

export function KeyValueItem({
  label,
  value,
  className,
  valueClassName,
}: {
  label: string;
  value: ReactNode;
  className?: string;
  valueClassName?: string;
}) {
  const displayValue = normalizeValue(value);

  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <dt className="text-xs font-semibold text-muted-foreground">{label}</dt>
      <dd className={cn("min-w-0 text-sm wrap-break-word text-foreground", valueClassName)}>
        {displayValue}
      </dd>
    </div>
  );
}

function normalizeValue(value: ReactNode) {
  if (value === null || value === undefined) return "-";
  if (typeof value === "string" && value.trim() === "") return "-";
  return value;
}
