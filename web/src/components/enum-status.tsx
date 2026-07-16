import { Status, StatusLabel } from "@/components/ui/status";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { StatusMetadataMap } from "@/lib/enum-metadata";

export function EnumStatus<T extends string>({
  value,
  metadata,
}: {
  value: T;
  metadata: StatusMetadataMap<T>;
}) {
  const item = metadata[value];
  const status = (
    <Status variant={item.variant ?? "default"}>
      <StatusLabel>{item.name}</StatusLabel>
    </Status>
  );

  if (!item.description) return status;

  return (
    <Tooltip>
      <TooltipTrigger render={status} />
      <TooltipContent className="max-w-72 text-left">{item.description}</TooltipContent>
    </Tooltip>
  );
}
