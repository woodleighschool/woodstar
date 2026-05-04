import * as React from "react";

import { cn } from "@/lib/utils";

export interface PageHeaderProps extends React.ComponentProps<"div"> {
  title: string;
  description?: React.ReactNode;
  actions?: React.ReactNode;
}

export function PageHeader({ title, description, actions, className, ...props }: PageHeaderProps) {
  return (
    <div
      className={cn("flex flex-col gap-2 border-b px-6 py-4 sm:flex-row sm:items-center sm:justify-between", className)}
      {...props}
    >
      <div className="min-w-0">
        <h1 className="text-lg font-semibold leading-tight truncate">{title}</h1>
        {description ? <p className="text-muted-foreground text-sm">{description}</p> : null}
      </div>
      {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
    </div>
  );
}
