import type { SVGProps } from "react";

import { PLATFORM_ICONS, platformIconTargetsLabel, type IconPath } from "@/components/platform/platform-icon-data";
import type { QueryablePlatform } from "@/lib/targeting";
import { cn } from "@/lib/utils";

export function PlatformIcon({
  platform,
  className,
  ...props
}: { platform: QueryablePlatform } & SVGProps<SVGSVGElement>) {
  return <SimpleIconGlyph icon={PLATFORM_ICONS[platform]} className={className} {...props} />;
}

export function SimpleIconGlyph({ icon, className, ...props }: { icon: IconPath } & SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={cn("size-4", className)} {...props}>
      <path d={icon.path} />
    </svg>
  );
}

export function PlatformIconList({
  platforms,
  className,
  iconClassName,
}: {
  platforms: readonly QueryablePlatform[];
  className?: string;
  iconClassName?: string;
}) {
  const label = platformIconTargetsLabel(platforms);
  return (
    <span className={cn("inline-flex items-center gap-1", className)} title={label}>
      {platforms.map((platform) => (
        <PlatformIcon key={platform} platform={platform} className={iconClassName} />
      ))}
    </span>
  );
}
