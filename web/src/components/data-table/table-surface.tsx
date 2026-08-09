import type { ComponentProps, ReactNode } from "react";

import { cn } from "@lib/utils";

export function TableSurface({
  heading,
  toolbar,
  empty,
  footer,
  variant = "flat",
  className,
  children,
  ...props
}: Omit<ComponentProps<"section">, "children"> & {
  heading?: ReactNode;
  toolbar?: ReactNode;
  empty?: ReactNode;
  footer?: ReactNode;
  variant?: "flat" | "embedded";
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={cn("flex w-full min-w-0 flex-col gap-3", className)} {...props}>
      {heading ? <h2 className="text-base/snug font-medium text-foreground">{heading}</h2> : null}
      {toolbar}
      <div
        className={cn(
          "flex min-w-0 flex-col overflow-hidden",
          variant === "flat" &&
            "rounded-xl bg-background text-foreground ring-1 ring-foreground/10",
        )}
      >
        {children}
        {empty != null ? (
          <output data-slot="table-surface-empty" className="block w-full min-w-0">
            {empty}
          </output>
        ) : null}
        {footer}
      </div>
    </section>
  );
}
