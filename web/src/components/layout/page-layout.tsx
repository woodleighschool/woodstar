import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { type ComponentProps, type ReactNode } from "react";

import { cn } from "@/lib/utils";

function PageShell({
  className,
  children,
  render,
  ...props
}: ComponentProps<"div"> & useRender.ComponentProps<"div">) {
  return useRender({
    defaultTagName: "div",
    props: mergeProps<"div">(
      {
        className: cn(
          "mx-auto flex w-full min-w-0 flex-col gap-6 px-4 py-5 sm:p-6 lg:px-8 lg:py-7",
          className,
        ),
        children,
      },
      props,
    ),
    render,
    state: {
      slot: "page-shell",
    },
  });
}

function PageHeader({
  title,
  description,
  actions,
  context,
  leading,
  className,
}: {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  context?: ReactNode;
  leading?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-wrap items-start justify-between gap-4", className)}>
      <div className="flex min-w-0 items-start gap-4">
        {leading ? <div className="shrink-0">{leading}</div> : null}
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2">
            <h1 className="text-xl font-semibold tracking-tight sm:text-2xl">{title}</h1>
            {context ? (
              <div className="flex min-w-0 flex-wrap items-center gap-2">{context}</div>
            ) : null}
          </div>
          {description ? (
            <p className="max-w-3xl text-sm text-muted-foreground">{description}</p>
          ) : null}
        </div>
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export { PageHeader, PageShell };
