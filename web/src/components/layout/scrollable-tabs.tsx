import { ScrollArea } from "@/components/ui/scroll-area";
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
    <ScrollArea orientation="horizontal">
      <TabsList
        variant={variant}
        className={cn(
          "w-max min-w-full justify-start [&_[data-slot=tabs-trigger]]:flex-none",
          variant === "line" && "gap-6 border-b px-0 pb-1.5 [&_[data-slot=tabs-trigger]]:px-0",
          className,
        )}
        {...props}
      />
    </ScrollArea>
  );
}
