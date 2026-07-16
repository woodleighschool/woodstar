import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { EnumMetadataMap } from "@/lib/enum-metadata";

export function EnumBadge<T extends string>({
  value,
  metadata,
}: {
  value: T;
  metadata: EnumMetadataMap<T>;
}) {
  const item = metadata[value];
  const badge = <Badge variant={item.variant ?? "secondary"}>{item.name}</Badge>;

  if (!item.description) return badge;

  return (
    <Tooltip>
      <TooltipTrigger render={badge} />
      <TooltipContent className="max-w-72 text-left">{item.description}</TooltipContent>
    </Tooltip>
  );
}
