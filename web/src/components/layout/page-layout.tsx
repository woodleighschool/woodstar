import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { type ComponentProps, type ReactNode } from "react";

import { cn } from "@lib/utils";

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
  icon,
  meta,
  className,
}: {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  context?: ReactNode;
  icon?: ReactNode;
  meta?: ReactNode;
  className?: string;
}) {
  const subtitleRows = Number(Boolean(description)) + Number(Boolean(meta));

  return (
    <div className={cn("flex flex-wrap items-start justify-between gap-4", className)}>
      <div
        className={cn(
          "grid min-w-0 gap-x-2 gap-y-1",
          icon &&
            (subtitleRows > 0
              ? "grid-cols-[2.5rem_minmax(0,1fr)] gap-x-3"
              : "grid-cols-[1.75rem_minmax(0,1fr)]"),
        )}
      >
        {icon ? (
          <span
            aria-hidden="true"
            className={cn(
              "inline-flex shrink-0 items-center justify-center self-center overflow-hidden rounded-md text-muted-foreground *:max-h-full *:max-w-full",
              subtitleRows > 0 ? "size-10 [&>svg]:size-6" : "size-7 [&>svg]:size-5",
              subtitleRows === 1 && "row-span-2",
              subtitleRows === 2 && "row-span-3",
            )}
          >
            {icon}
          </span>
        ) : null}
        <div
          className={cn(
            "flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2",
            icon && "col-start-2",
          )}
        >
          <h1 className="min-w-0 text-xl font-semibold tracking-tight sm:text-2xl">
            <span className="min-w-0 wrap-break-word">{title}</span>
          </h1>
          {context ? (
            <div className="flex min-w-0 flex-wrap items-center gap-2">{context}</div>
          ) : null}
        </div>
        {description ? (
          <p className={cn("max-w-3xl text-sm text-muted-foreground", icon && "col-start-2")}>
            {description}
          </p>
        ) : null}
        {meta ? (
          <div className={cn("text-sm text-muted-foreground", icon && "col-start-2")}>{meta}</div>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export { PageHeader, PageShell };
