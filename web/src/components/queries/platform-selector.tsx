import { Apple, Globe, Monitor, Server } from "lucide-react";
import type { ComponentType, SVGProps } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  PLATFORM_LABELS,
  QUERYABLE_PLATFORMS,
  platformsFromValue,
  platformsToValue,
  type QueryablePlatform,
} from "@/lib/targeting";
import { cn } from "@/lib/utils";

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
  const selected = new Set(platformsFromValue(value));
  const allSelected = selected.size === 0;

  function setAll() {
    onChange(undefined);
  }

  function toggle(platform: QueryablePlatform, checked: boolean) {
    const next = new Set(selected);
    if (checked) next.add(platform);
    else next.delete(platform);
    onChange(platformsToValue([...next]));
  }

  return (
    <div className="grid gap-2">
      <Label>Targeted platforms</Label>
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={disabled}
          data-selected={allSelected}
          className={cn(
            "border-input hover:bg-muted inline-flex h-9 items-center rounded-md border px-3 text-sm",
            "data-[selected=true]:bg-primary data-[selected=true]:text-primary-foreground data-[selected=true]:border-primary",
          )}
          onClick={setAll}
        >
          All platforms
        </button>
        {QUERYABLE_PLATFORMS.map((platform) => {
          const Icon = ICONS[platform];
          const checked = selected.has(platform);
          return (
            <label
              key={platform}
              className={cn(
                "border-input hover:bg-muted inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm",
                checked && "border-primary bg-primary text-primary-foreground",
              )}
            >
              <Checkbox
                checked={checked}
                disabled={disabled}
                className={checked ? "border-primary-foreground" : undefined}
                onCheckedChange={(next) => toggle(platform, next === true)}
              />
              <Icon className="size-4" />
              {PLATFORM_LABELS[platform]}
            </label>
          );
        })}
      </div>
    </div>
  );
}
