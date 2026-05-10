import { Apple, Globe, Monitor, Server } from "lucide-react";
import type { ComponentType, SVGProps } from "react";

import { Label } from "@/components/ui/label";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  PLATFORM_LABELS,
  QUERYABLE_PLATFORMS,
  platformsFromValue,
  platformsToValue,
  type QueryablePlatform,
} from "@/lib/targeting";

const ICONS: Record<QueryablePlatform, ComponentType<SVGProps<SVGSVGElement>>> = {
  darwin: Apple,
  windows: Monitor,
  linux: Server,
  chrome: Globe,
};

export function PlatformSelector({
  value,
  onChange,
  disabled = false,
}: {
  value?: string | null;
  onChange: (next: string | undefined) => void;
  disabled?: boolean;
}) {
  const selected = platformsFromValue(value);

  function handleChange(next: string[]) {
    onChange(platformsToValue(next as QueryablePlatform[]));
  }

  return (
    <div className="grid gap-2">
      <Label>Targeted platforms</Label>
      <ToggleGroup
        type="multiple"
        value={selected}
        onValueChange={handleChange}
        disabled={disabled}
        variant="outline"
        className="flex-wrap justify-start"
      >
        {QUERYABLE_PLATFORMS.map((platform) => {
          const Icon = ICONS[platform];
          return (
            <ToggleGroupItem key={platform} value={platform} className="gap-2">
              <Icon className="size-4" />
              {PLATFORM_LABELS[platform]}
            </ToggleGroupItem>
          );
        })}
      </ToggleGroup>
      <p className="text-muted-foreground text-xs">
        {selected.length === 0 ? "Targeting all platforms." : "Toggle to limit which platforms run this."}
      </p>
    </div>
  );
}
