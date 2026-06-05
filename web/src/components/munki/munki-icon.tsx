import { SoftwareIcon, type SoftwareIconSize } from "@/components/software/software-icon";
import { cn } from "@/lib/utils";

const IMAGE_SIZE_CLASS = {
  sm: "size-6 rounded-sm",
  md: "size-9 rounded-md",
  lg: "size-20 rounded-lg",
} as const;

interface MunkiIconProps {
  iconUrl?: string;
  fallbackIconUrl?: string;
  size?: SoftwareIconSize;
  className?: string;
  loading?: "eager" | "lazy";
}

export function MunkiIcon({ iconUrl, fallbackIconUrl, size = "sm", className, loading = "lazy" }: MunkiIconProps) {
  const url = iconUrl ?? fallbackIconUrl;
  if (!url) {
    return <SoftwareIcon source="pkg_packages" size={size} className={className} />;
  }

  return (
    <span className={cn("bg-muted/40 inline-flex shrink-0 overflow-hidden", IMAGE_SIZE_CLASS[size], className)}>
      <img src={url} alt="" className="size-full object-contain" loading={loading} />
    </span>
  );
}

export type { SoftwareIconSize as MunkiIconSize };
