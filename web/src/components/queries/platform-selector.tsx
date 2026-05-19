import { PlatformIcon } from "@/components/platform/platform-icons";
import { Label } from "@/components/ui/label";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  PLATFORM_LABELS,
  QUERYABLE_PLATFORMS,
  platformsFromValue,
  platformsToValue,
  type QueryablePlatform,
} from "@/lib/targeting";

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
      <PlatformToggleGroup selected={selected} onChange={handleChange} disabled={disabled} />
      <p className="text-muted-foreground text-xs">
        {selected.length === 0 ? "Targeting all platforms." : "Toggle to limit which platforms run this."}
      </p>
    </div>
  );
}

export function PlatformToggleGroup({
  selected,
  onChange,
  disabled = false,
}: {
  selected: QueryablePlatform[];
  onChange: (next: QueryablePlatform[]) => void;
  disabled?: boolean;
}) {
  return (
    <ToggleGroup
      type="multiple"
      value={selected}
      onValueChange={(next) => onChange(next as QueryablePlatform[])}
      disabled={disabled}
      variant="outline"
      className="flex-wrap justify-start"
    >
      {QUERYABLE_PLATFORMS.map((platform) => (
        <ToggleGroupItem key={platform} value={platform} className="gap-2">
          <PlatformIcon platform={platform} className="size-4" />
          {PLATFORM_LABELS[platform]}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  );
}
