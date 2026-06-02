import { SoftwareIcon, type SoftwareIconSize } from "@/components/software/software-icon";
import { cn } from "@/lib/utils";

const IMAGE_SIZE_CLASS = {
  sm: { box: "size-6 rounded-sm", image: "p-0.5" },
  md: { box: "size-9 rounded-md", image: "p-1" },
  lg: { box: "size-20 rounded-lg", image: "p-2" },
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

  const sizes = IMAGE_SIZE_CLASS[size];
  return (
    <span
      className={cn(
        "bg-muted/40 inline-flex shrink-0 items-center justify-center overflow-hidden border",
        sizes.box,
        className,
      )}
    >
      <img src={url} alt="" className={cn("size-full object-contain", sizes.image)} loading={loading} />
    </span>
  );
}

export type { SoftwareIconSize as MunkiIconSize };
