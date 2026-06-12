import type { ComponentProps } from "react";

import type { Badge } from "@/components/ui/badge";
import type { Status } from "@/components/ui/status";

type BadgeVariant = ComponentProps<typeof Badge>["variant"];
type StatusVariant = ComponentProps<typeof Status>["variant"];

interface EnumLabel {
  name: string;
  description?: string;
}

export interface EnumMetadata extends EnumLabel {
  variant?: BadgeVariant;
}

export interface StatusMetadata extends EnumLabel {
  variant?: StatusVariant;
}

export type EnumMetadataMap<T extends string> = Record<T, EnumMetadata>;
export type StatusMetadataMap<T extends string> = Record<T, StatusMetadata>;

export function enumOptions<T extends string>(
  metadata: Record<T, EnumLabel>,
): Array<{ value: T; label: string }> {
  return (Object.entries(metadata) as Array<[T, EnumLabel]>).map(([value, item]) => ({
    value,
    label: item.name,
  }));
}

export function enumLabel<T extends string>(
  metadata: Partial<Record<T, EnumLabel>>,
  value: T | string | null | undefined,
  fallback = "Unknown",
) {
  if (!value) return fallback;
  return metadata[value as T]?.name ?? fallback;
}
