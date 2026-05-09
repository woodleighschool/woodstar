import { Tabs as TabsPrimitive } from "radix-ui";
import * as React from "react";

import { cn } from "@/lib/utils";

function PageTabs({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return <TabsPrimitive.Root data-slot="page-tabs" className={cn("flex flex-col gap-4", className)} {...props} />;
}

function PageTabsList({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      data-slot="page-tabs-list"
      className={cn("inline-flex w-fit items-center gap-1", className)}
      {...props}
    />
  );
}

function PageTabsTrigger({ className, children, ...props }: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      data-slot="page-tabs-trigger"
      className={cn(
        // base
        "text-muted-foreground inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm transition-colors",
        // hover
        "hover:bg-muted/60 hover:text-foreground",
        // focus
        "focus-visible:ring-ring/50 focus-visible:outline-none focus-visible:ring-2",
        // active
        "data-[state=active]:bg-muted data-[state=active]:text-foreground data-[state=active]:font-semibold",
        className,
      )}
      {...props}
    >
      {/* Reserve space for the bold variant so toggling state doesn't shift layout. */}
      <span
        className="relative inline-block before:invisible before:block before:h-0 before:overflow-hidden before:font-semibold before:content-[attr(data-text)]"
        data-text={typeof children === "string" ? children : undefined}
      >
        {children}
      </span>
    </TabsPrimitive.Trigger>
  );
}

function PageTabsContent({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="page-tabs-content"
      className={cn("flex flex-col gap-4 outline-none", className)}
      {...props}
    />
  );
}

export { PageTabs, PageTabsContent, PageTabsList, PageTabsTrigger };
