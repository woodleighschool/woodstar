import { Status, StatusLabel } from "@/components/ui/status";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { StatusMetadata, StatusMetadataMap } from "@/lib/enum-metadata";

const FALLBACK: StatusMetadata = {
  name: "Unknown",
  description: "Woodstar does not recognize this value yet.",
  variant: "default",
};

export function EnumStatus<T extends string>({
  value,
  metadata,
  fallback = FALLBACK,
}: {
  value: T | string;
  metadata: Partial<StatusMetadataMap<T>>;
  fallback?: StatusMetadata;
}) {
  const item = metadata[value as T] ?? fallback;
  const status = (
    <Status variant={item.variant ?? "default"}>
      <StatusLabel>{item.name}</StatusLabel>
    </Status>
  );

  if (!item.description) return status;

  return (
    <Tooltip>
      <TooltipTrigger asChild>{status}</TooltipTrigger>
      <TooltipContent className="max-w-72 text-left">{item.description}</TooltipContent>
    </Tooltip>
  );
}
