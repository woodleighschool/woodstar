import { type ReactNode, useState } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTabIndicator } from "@/hooks/use-tab-indicator";
import { cn } from "@/lib/utils";

export interface ScrollableTab {
  value: string;
  label: string;
  content: ReactNode;
  disabled?: boolean;
}

// App-standard line tabs that scroll horizontally when they outgrow the row.
// The scroll/underline styling has to live here because ui/tabs is vendored.
export function ScrollableTabs({
  tabs,
  defaultValue = tabs[0]?.value,
  className,
}: {
  tabs: ScrollableTab[];
  defaultValue?: string;
  className?: string;
}) {
  const [value, setValue] = useState(defaultValue);
  const { listRef, box } = useTabIndicator(value);

  return (
    <Tabs value={value} onValueChange={setValue} className={cn("gap-5", className)}>
      <ScrollArea orientation="horizontal">
        <TabsList
          ref={listRef}
          variant="line"
          className="relative w-max min-w-full justify-start gap-6 border-b px-0 pb-1.5"
        >
          <span
            aria-hidden
            className="pointer-events-none absolute bottom-0 left-0 h-0.5 rounded-full bg-foreground transition-[transform,width,opacity] duration-300 ease-out"
            style={{
              transform: `translateX(${box?.left ?? 0}px)`,
              width: box?.width ?? 0,
              opacity: box ? 1 : 0,
            }}
          />
          {tabs.map((tab) => (
            <TabsTrigger
              key={tab.value}
              value={tab.value}
              disabled={tab.disabled}
              className="flex-none px-0 after:hidden"
            >
              {tab.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </ScrollArea>
      {tabs.map((tab) => (
        <TabsContent key={tab.value} value={tab.value} className="min-w-0">
          {tab.content}
        </TabsContent>
      ))}
    </Tabs>
  );
}
