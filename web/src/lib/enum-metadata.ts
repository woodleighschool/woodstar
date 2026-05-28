import type { ComponentProps } from "react";

import type { Badge } from "@/components/ui/badge";

type BadgeVariant = ComponentProps<typeof Badge>["variant"];

export interface EnumMetadata {
  name: string;
  description?: string;
  variant?: BadgeVariant;
  indicatorClassName?: string;
}

export type EnumMetadataMap<T extends string> = Record<T, EnumMetadata>;

export function enumOptions<T extends string>(metadata: EnumMetadataMap<T>): Array<{ value: T; label: string }> {
  return (Object.entries(metadata) as Array<[T, EnumMetadata]>).map(([value, item]) => ({
    value,
    label: item.name,
  }));
}

export function enumLabel<T extends string>(
  metadata: Partial<Record<T, EnumMetadata>>,
  value: T | string | null | undefined,
  fallback = "Unknown",
) {
  if (!value) return fallback;
  return metadata[value as T]?.name ?? fallback;
}
