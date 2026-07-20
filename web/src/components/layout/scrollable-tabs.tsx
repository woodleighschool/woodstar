import { ScrollArea as ScrollAreaPrimitive } from "@base-ui/react/scroll-area";

import { ScrollBar } from "@/components/ui/scroll-area";
import { Tabs, TabsList } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

export function ScrollableTabs({ className, ...props }: React.ComponentProps<typeof Tabs>) {
  return <Tabs className={cn("gap-5", className)} {...props} />;
}

export function ScrollableTabsList({
  className,
  variant = "line",
  ...props
}: React.ComponentProps<typeof TabsList>) {
  return (
    <ScrollAreaPrimitive.Root className="relative w-full whitespace-nowrap">
      <ScrollAreaPrimitive.Viewport className="size-full">
        <TabsList
          variant={variant}
          className={cn(
            `
              w-max justify-start
              **:data-[slot=tabs-trigger]:flex-none
            `,
            variant === "line" &&
              `
                min-w-full gap-6 border-b px-0 pb-1.5
                **:data-[slot=tabs-trigger]:px-0
              `,
            className,
          )}
          {...props}
        />
      </ScrollAreaPrimitive.Viewport>
      <ScrollBar orientation="horizontal" />
    </ScrollAreaPrimitive.Root>
  );
}
