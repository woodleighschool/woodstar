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

export function enumOptions<const Value extends string>(
  metadata: Record<Value, EnumLabel>,
  values: readonly Value[],
): Array<{ value: Value; label: string }> {
  return values.map((value) => ({
    value,
    label: metadata[value].name,
  }));
}

export function enumLabel<T extends string>(metadata: Record<T, EnumLabel>, value: T): string {
  return metadata[value].name;
}
