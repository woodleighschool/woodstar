import { type ReactNode } from "react";

import { cn } from "@/lib/utils";

export function TargetSection({
  title,
  action,
  children,
  className,
  contentClassName,
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <section className={cn("flex flex-col gap-3", className)}>
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{title}</h2>
        {action ? <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">{action}</div> : null}
      </div>
      <div className={cn("flex flex-col gap-3", contentClassName)}>{children}</div>
    </section>
  );
}
