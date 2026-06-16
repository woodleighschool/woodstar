import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export interface KeyValue {
  label: string;
  value: ReactNode;
  className?: string;
  valueClassName?: string;
}

export function KeyValueGrid({
  items,
  className,
  children,
}: {
  items?: KeyValue[];
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
      {items?.map((item) => (
        <KeyValueItem
          key={String(item.label)}
          label={item.label}
          value={item.value}
          className={item.className}
          valueClassName={item.valueClassName}
        />
      ))}
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
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <dt className="text-xs font-semibold text-muted-foreground">{label}</dt>
      <dd className={cn("min-w-0 text-sm break-words text-foreground", valueClassName)}>{value}</dd>
    </div>
  );
}
