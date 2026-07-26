import type { ReactNode } from "react";

import { Separator } from "@components/ui/separator";
import { cn } from "@lib/utils";

export function KeyValueRows({
  className,
  children,
}: {
  className?: string;
  children?: ReactNode;
}) {
  return <dl className={cn("min-w-0", className)}>{children}</dl>;
}

export function KeyValueRow({
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
    <div
      className={cn(
        "grid min-w-0 gap-1 rounded-md px-4 py-3 even:bg-muted/50 sm:grid-cols-[minmax(12rem,16rem)_minmax(0,1fr)] sm:gap-6",
        className,
      )}
    >
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className={cn("min-w-0 text-sm wrap-break-word text-foreground", valueClassName)}>
        {displayValue}
      </dd>
    </div>
  );
}

export function KeyValueSection({
  title,
  actions,
  className,
  children,
}: {
  title: string;
  actions?: ReactNode;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <section className={cn("flex min-w-0 flex-col gap-3", className)}>
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base/snug font-medium text-foreground">{title}</h2>
        {actions}
      </div>
      <Separator />
      <KeyValueRows>{children}</KeyValueRows>
    </section>
  );
}

function normalizeValue(value: ReactNode) {
  if (value === null || value === undefined) return "-";
  if (typeof value === "string" && value.trim() === "") return "-";
  return value;
}
