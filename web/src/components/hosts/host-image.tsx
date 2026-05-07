import { Laptop } from "lucide-react";

import { useAppleDbImage } from "@/hooks/use-appledb-image";
import { cn } from "@/lib/utils";

/**
 * Container that shows either the appledb hero image for the
 * given hardware model or a laptop placeholder.
 */
export function HostImage({
  hardwareModel,
  className,
}: {
  hardwareModel: string | null | undefined;
  className?: string;
}) {
  const url = useAppleDbImage(hardwareModel);
  return (
    <div
      className={cn(
        "bg-muted/40 flex size-20 shrink-0 items-center justify-center overflow-hidden rounded-lg border",
        className,
      )}
    >
      {url ? (
        <img src={url} alt="" loading="lazy" className="size-full object-contain p-1" />
      ) : (
        <Laptop className="text-muted-foreground size-8" aria-hidden />
      )}
    </div>
  );
}
