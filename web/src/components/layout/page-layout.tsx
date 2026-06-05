import { cloneElement, isValidElement, type ComponentProps, type ReactElement, type ReactNode } from "react";

import { AppBreadcrumbs } from "@/components/layout/app-breadcrumbs";
import { cn } from "@/lib/utils";

function PageShell({
  className,
  asChild = false,
  children,
  ...props
}: ComponentProps<"div"> & {
  asChild?: boolean;
}) {
  const shellClassName = cn("flex min-w-0 flex-col gap-5 p-6", className);
  const shellChildren = (
    <>
      <AppBreadcrumbs className="-mb-1" />
      {children}
    </>
  );

  if (asChild && isValidElement(children)) {
    const child = children as ReactElement<{ className?: string; children?: ReactNode }>;

    return cloneElement(child, {
      ...props,
      className: cn(shellClassName, child.props.className),
      children: (
        <>
          <AppBreadcrumbs className="-mb-1" />
          {child.props.children}
        </>
      ),
    });
  }

  return (
    <div className={shellClassName} {...props}>
      {shellChildren}
    </div>
  );
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
            <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
            {context ? <div className="flex min-w-0 flex-wrap items-center gap-2">{context}</div> : null}
          </div>
          {description ? <p className="text-muted-foreground max-w-3xl text-sm">{description}</p> : null}
        </div>
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export { PageHeader, PageShell };
