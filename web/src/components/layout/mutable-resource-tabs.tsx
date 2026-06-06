import type { ReactNode } from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

export interface MutableResourceTab {
  value: string;
  label: string;
  content: ReactNode;
  disabled?: boolean;
}

export function MutableResourceTabs({
  tabs,
  defaultValue = tabs[0]?.value,
  className,
}: {
  tabs: MutableResourceTab[];
  defaultValue?: string;
  className?: string;
}) {
  return (
    <Tabs defaultValue={defaultValue} className={cn("gap-5", className)}>
      <TabsList variant="line" className="w-full justify-start gap-6 border-b px-0">
        {tabs.map((tab) => (
          <TabsTrigger key={tab.value} value={tab.value} disabled={tab.disabled} className="flex-none px-0">
            {tab.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {tabs.map((tab) => (
        <TabsContent key={tab.value} value={tab.value} className="min-w-0">
          {tab.content}
        </TabsContent>
      ))}
    </Tabs>
  );
}
