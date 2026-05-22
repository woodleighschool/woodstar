import { Star } from "lucide-react";

import { cn } from "@/lib/utils";

type WoodstarMarkSize = "sm" | "md";

interface WoodstarMarkProps {
  className?: string;
  size?: WoodstarMarkSize;
}

const sizeClass: Record<WoodstarMarkSize, string> = {
  sm: "size-8",
  md: "size-10",
};

const iconClass: Record<WoodstarMarkSize, string> = {
  sm: "size-6",
  md: "size-8",
};

export function WoodstarMark({ className, size = "sm" }: WoodstarMarkProps) {
  return (
    <div
      aria-hidden="true"
      className={cn("relative flex shrink-0 items-center justify-center text-primary", sizeClass[size], className)}
    >
      <Star className={iconClass[size]} />
    </div>
  );
}
