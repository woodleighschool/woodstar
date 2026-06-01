import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { EnumMetadata, EnumMetadataMap } from "@/lib/enum-metadata";

const FALLBACK: EnumMetadata = {
  name: "Unknown",
  description: "Woodstar does not recognize this value yet.",
  variant: "secondary",
};

export function EnumBadge<T extends string>({
  value,
  metadata,
  fallback = FALLBACK,
}: {
  value: T | string;
  metadata: Partial<EnumMetadataMap<T>>;
  fallback?: EnumMetadata;
}) {
  const item = metadata[value as T] ?? fallback;
  const badge = <Badge variant={item.variant ?? "secondary"}>{item.name}</Badge>;

  if (!item.description) return badge;

  return (
    <Tooltip>
      <TooltipTrigger asChild>{badge}</TooltipTrigger>
      <TooltipContent className="max-w-72 text-left">{item.description}</TooltipContent>
    </Tooltip>
  );
}
