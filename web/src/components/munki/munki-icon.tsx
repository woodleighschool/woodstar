import { Package } from "lucide-react";

import type { SoftwareIconSize } from "@/components/software/software-icon";
import { cn } from "@/lib/utils";

const ICON_SIZE_CLASS = {
  sm: "size-6 rounded-sm",
  md: "size-9 rounded-md",
  lg: "size-20 rounded-lg",
} as const;

const FALLBACK_ICON_CLASS = {
  sm: "size-4",
  md: "size-5",
  lg: "size-8",
} as const;

interface MunkiIconProps {
  iconUrl?: string;
  size?: SoftwareIconSize;
  className?: string;
  loading?: "eager" | "lazy";
}

export function MunkiIcon({ iconUrl, size = "sm", className, loading = "lazy" }: MunkiIconProps) {
  if (!iconUrl) {
    return (
      <span
        className={cn(
          "bg-muted/40 text-muted-foreground inline-flex shrink-0 items-center justify-center",
          ICON_SIZE_CLASS[size],
          className,
        )}
      >
        <Package className={FALLBACK_ICON_CLASS[size]} />
      </span>
    );
  }

  return (
    <img
      src={iconUrl}
      alt=""
      className={cn("block shrink-0 object-contain", ICON_SIZE_CLASS[size], className)}
      loading={loading}
    />
  );
}

export type { SoftwareIconSize as MunkiIconSize };
